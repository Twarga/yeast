// Package daemon hosts the local yeastd service: an HTTP API over a
// user-private unix socket (0600) with the exact same app.Service and state
// locks the CLI uses, plus SSE events and a WebSocket-to-VNC proxy.
package daemon

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Twarga/yeast/internal/app"
	"github.com/Twarga/yeast/internal/project"
)

// Options configures the daemon server.
type Options struct {
	SocketPath  string        // unix socket path (default $XDG_RUNTIME_DIR/yeast/yeastd.sock)
	TCPAddr     string        // optional loopback TCP listener (requires Token)
	Token       string        // bearer token for the TCP listener
	IdleTimeout time.Duration // exit when no requests arrive for this long (0 = never)
	YeastHome   string        // override Yeast home (tests)
}

// Server is the yeastd HTTP service.
type Server struct {
	opts    Options
	service *app.Service
	reg     *Registry
	mux     *http.ServeMux

	hubsMu sync.Mutex
	hubs   map[string]*Hub

	mcpServers   sync.Map // root -> *mcp.Server
	lastActivity atomic.Int64 // unix nano
	activeConns  atomic.Int64 // long-lived SSE / WebSocket connections
}

// New builds the daemon. service may be nil → a fresh app.Service is made.
func New(service *app.Service, opts Options) (*Server, error) {
	if service == nil {
		service = app.NewService()
	}
	yeastHome := opts.YeastHome
	if yeastHome == "" {
		var err error
		yeastHome, err = project.DefaultYeastHome()
		if err != nil {
			return nil, err
		}
	}
	reg, err := NewRegistry(yeastHome)
	if err != nil {
		return nil, err
	}
	if opts.SocketPath == "" {
		opts.SocketPath = DefaultSocketPath()
	}
	if opts.TCPAddr != "" && opts.Token == "" {
		return nil, fmt.Errorf("--token is required when --tcp is set")
	}
	s := &Server{
		opts:    opts,
		service: service,
		reg:     reg,
		hubs:    map[string]*Hub{},
	}
	s.lastActivity.Store(time.Now().UnixNano())
	s.routes()
	return s, nil
}

// DefaultSocketPath returns $XDG_RUNTIME_DIR/yeast/yeastd.sock (falling back
// to /tmp/yeast-<uid>/yeastd.sock when XDG is unset).
func DefaultSocketPath() string {
	if dir := os.Getenv("XDG_RUNTIME_DIR"); dir != "" {
		return filepath.Join(dir, "yeast", "yeastd.sock")
	}
	return filepath.Join(os.TempDir(), fmt.Sprintf("yeast-%d", os.Getuid()), "yeastd.sock")
}

// SocketPath exposes the bound unix socket path.
func (s *Server) SocketPath() string { return s.opts.SocketPath }

// touch records activity for the idle watchdog.
func (s *Server) touch() { s.lastActivity.Store(time.Now().UnixNano()) }

func (s *Server) hub(id string) *Hub {
	s.hubsMu.Lock()
	defer s.hubsMu.Unlock()
	h, ok := s.hubs[id]
	if !ok {
		h = newHub()
		s.hubs[id] = h
	}
	return h
}

// routes wires the API surface.
func (s *Server) routes() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("POST /v1/projects", s.handleRegisterProject)
	mux.HandleFunc("GET /v1/projects", s.handleListProjects)
	mux.HandleFunc("GET /v1/projects/{id}/status", s.handleStatus)
	mux.HandleFunc("GET /v1/projects/{id}/events", s.handleEvents)
	mux.HandleFunc("POST /v1/projects/{id}/up", s.handleUp)
	mux.HandleFunc("POST /v1/projects/{id}/down", s.handleDown)
	mux.HandleFunc("POST /v1/projects/{id}/exec", s.handleExec)
	mux.HandleFunc("POST /v1/projects/{id}/snapshot", s.handleSnapshot)
	mux.HandleFunc("POST /v1/projects/{id}/restore", s.handleRestore)
	mux.HandleFunc("POST /v1/projects/{id}/fork", s.handleFork)
	mux.HandleFunc("POST /v1/projects/{id}/screenshot", s.handleScreenshot)
	mux.HandleFunc("POST /v1/projects/{id}/input", s.handleInput)
	mux.HandleFunc("POST /v1/projects/{id}/export", s.handleExport)
	mux.HandleFunc("POST /v1/projects/{id}/clean", s.handleClean)
	mux.HandleFunc("POST /v1/projects/{id}/destroy", s.handleDestroy)
	mux.HandleFunc("GET /v1/projects/{id}/snapshots", s.handleSnapshots)
	mux.HandleFunc("GET /v1/projects/{id}/logs", s.handleLogs)
	mux.HandleFunc("GET /v1/projects/{id}/instances/{name}/vnc", s.handleVNC)
	mux.HandleFunc("POST /mcp", s.handleMCP)
	s.mux = mux
}

// Handler exposes the mux for in-process tests.
func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.touch()
		r = r.WithContext(app.WithActor(r.Context(), "api"))
		s.mux.ServeHTTP(w, r)
	})
}

// Serve listens on the unix socket (mode 0600) and optional TCP address,
// reattaches to running VMs, and blocks until ctx is cancelled or the idle
// timeout elapses.
func (s *Server) Serve(ctx context.Context) error {
	if err := s.reattach(ctx); err != nil {
		return fmt.Errorf("reattach: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(s.opts.SocketPath), 0700); err != nil {
		return err
	}
	_ = os.Remove(s.opts.SocketPath)
	unixLn, err := net.Listen("unix", s.opts.SocketPath)
	if err != nil {
		return fmt.Errorf("listen unix: %w", err)
	}
	defer unixLn.Close()
	if err := os.Chmod(s.opts.SocketPath, 0600); err != nil {
		return err
	}

	httpServer := &http.Server{Handler: s.Handler()}
	errCh := make(chan error, 2)
	go func() {
		if err := httpServer.Serve(unixLn); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	var tcpLn net.Listener
	var tcpServer *http.Server
	if s.opts.TCPAddr != "" {
		tcpLn, err = net.Listen("tcp", s.opts.TCPAddr)
		if err != nil {
			return fmt.Errorf("listen tcp: %w", err)
		}
		defer tcpLn.Close()
		tcpServer = &http.Server{Handler: s.bearerMiddleware(s.Handler())}
		go func() {
			if err := tcpServer.Serve(tcpLn); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errCh <- err
			}
		}()
	}

	shutdownAll := func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
		if tcpServer != nil {
			_ = tcpServer.Shutdown(shutdownCtx)
		}
	}

	// Idle watchdog
	var idle <-chan time.Time
	var idleTimer *time.Timer
	if s.opts.IdleTimeout > 0 {
		idleTimer = time.NewTimer(s.opts.IdleTimeout)
		defer idleTimer.Stop()
		idle = idleTimer.C
	}

	for {
		select {
		case <-ctx.Done():
			shutdownAll()
			return nil
		case err := <-errCh:
			shutdownAll()
			return err
		case <-idle:
			if s.activeConns.Load() == 0 && time.Since(time.Unix(0, s.lastActivity.Load())) >= s.opts.IdleTimeout {
				shutdownAll()
				return nil
			}
			idleTimer.Reset(s.opts.IdleTimeout)
		}
	}
}

// reattach walks registered projects and reconciles each state file with
// live QEMU processes — daemon restarts never kill running VMs because QEMU
// processes are independent of this process.
func (s *Server) reattach(ctx context.Context) error {
	for id, root := range s.reg.List() {
		if err := s.service.ReconcileProject(ctx, root); err != nil {
			// Root gone or unreadable — drop the stale registration.
			_ = s.reg.Forget(id)
		}
	}
	return nil
}

// bearerMiddleware guards the optional TCP listener.
// Browser WebSocket clients (e.g. noVNC) cannot set custom Authorization headers,
// so ?token=<bearer> is also accepted on /vnc upgrades.
func (s *Server) bearerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		want := "Bearer " + s.opts.Token
		got := r.Header.Get("Authorization")
		authorized := subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
		if !authorized && filepath.Base(r.URL.Path) == "vnc" {
			if qTok := r.URL.Query().Get("token"); qTok != "" {
				authorized = subtle.ConstantTimeCompare([]byte(qTok), []byte(s.opts.Token)) == 1
			}
		}
		if !authorized {
			http.Error(w, `{"ok":false,"error":{"code":"unauthorized","message":"invalid or missing bearer token"}}`, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
