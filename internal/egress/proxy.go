// Package egress implements Yeast's lightweight HTTP/CONNECT forward proxy
// for egress.mode=allowlist. Guests route through it via QEMU guestfwd;
// every allowed/denied domain decision is logged.
package egress

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// Proxy is a minimal forward proxy enforcing a domain allowlist.
type Proxy struct {
	allowlist map[string]bool
	logPath   string
	mu        sync.Mutex
	logFile   *os.File
}

// NewProxy builds a proxy; logPath receives one line per decision
// (timestamp, client, host, ALLOW/DENY) — `yeast logs <vm> --network`
// surfaces it.
func NewProxy(allowlist []string, logPath string) (*Proxy, error) {
	p := &Proxy{allowlist: map[string]bool{}, logPath: logPath}
	for _, d := range allowlist {
		p.allowlist[strings.ToLower(strings.TrimSpace(d))] = true
	}
	if logPath != "" {
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			return nil, err
		}
		p.logFile = f
	}
	return p, nil
}

func (p *Proxy) Close() {
	if p.logFile != nil {
		_ = p.logFile.Close()
	}
}

// allowed reports whether host (with or without port) is on the allowlist.
func (p *Proxy) allowed(host string) bool {
	host = strings.ToLower(host)
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	if p.allowlist[host] {
		return true
	}
	// suffix match: allow "example.com" to cover "api.example.com"
	for d := range p.allowlist {
		if strings.HasSuffix(host, "."+d) {
			return true
		}
	}
	return false
}

func (p *Proxy) log(client, host, verdict string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.logFile == nil {
		return
	}
	fmt.Fprintf(p.logFile, "%s %s %s %s\n",
		time.Now().UTC().Format(time.RFC3339), client, verdict, host)
}

// ServeHTTP implements http.Handler: CONNECT → tunnel, plain → forward.
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		p.handleConnect(w, r)
		return
	}
	host := r.URL.Host
	if host == "" {
		host = r.Host
	}
	if !p.allowed(host) {
		p.log(r.RemoteAddr, host, "DENY")
		http.Error(w, "yeast egress proxy: domain not in allowlist", http.StatusForbidden)
		return
	}
	p.log(r.RemoteAddr, host, "ALLOW")
	p.forward(w, r, host)
}

func (p *Proxy) handleConnect(w http.ResponseWriter, r *http.Request) {
	if !p.allowed(r.Host) {
		p.log(r.RemoteAddr, r.Host, "DENY")
		http.Error(w, "yeast egress proxy: domain not in allowlist", http.StatusForbidden)
		return
	}
	p.log(r.RemoteAddr, r.Host, "ALLOW")
	upstream, err := net.DialTimeout("tcp", ensurePort(r.Host, "443"), 10*time.Second)
	if err != nil {
		http.Error(w, "upstream unreachable", http.StatusBadGateway)
		return
	}
	hj, ok := w.(http.Hijacker)
	if !ok {
		upstream.Close()
		http.Error(w, "hijack unsupported", http.StatusInternalServerError)
		return
	}
	conn, brw, err := hj.Hijack()
	if err != nil {
		upstream.Close()
		return
	}
	_, _ = brw.WriteString("HTTP/1.1 200 Connection Established\r\n\r\n")
	_ = brw.Flush()
	go pipeBoth(conn, upstream)
}

func (p *Proxy) forward(w http.ResponseWriter, r *http.Request, host string) {
	outReq := r.Clone(r.Context())
	outReq.RequestURI = ""
	outReq.URL.Scheme = "http"
	outReq.URL.Host = ensurePort(host, "80")
	outReq.Host = host
	resp, err := http.DefaultTransport.RoundTrip(outReq)
	if err != nil {
		http.Error(w, "upstream error", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func ensurePort(host, def string) string {
	if _, _, err := net.SplitHostPort(host); err == nil {
		return host
	}
	return net.JoinHostPort(host, strings.TrimPrefix(def, ":"))
}

func pipeBoth(a, b net.Conn) {
	defer a.Close()
	defer b.Close()
	done := make(chan struct{}, 2)
	go func() { _, _ = io.Copy(a, b); done <- struct{}{} }()
	go func() { _, _ = io.Copy(b, a); done <- struct{}{} }()
	<-done
}

// Serve runs the proxy on a listener until ctx ends. Used by the
// `yeast egress-proxy` sidecar and tests.
func Serve(ctx context.Context, ln net.Listener, p *Proxy) error {
	srv := &http.Server{Handler: p, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		<-ctx.Done()
		_ = srv.Close()
	}()
	return srv.Serve(ln)
}
