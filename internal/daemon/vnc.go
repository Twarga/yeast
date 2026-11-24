package daemon

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/coder/websocket"

	"github.com/Twarga/yeast/internal/app"
	"github.com/Twarga/yeast/internal/project"
	"github.com/Twarga/yeast/internal/state"
)

// handleVNC upgrades the connection to WebSocket and pipes binary frames to
// the instance's loopback VNC TCP port — a built-in websockify replacement.
func (s *Server) handleVNC(w http.ResponseWriter, r *http.Request) {
	root, ok := s.projectRoot(w, r)
	if !ok {
		return
	}
	name := r.PathValue("name")
	host, port, err := s.displayEndpoint(root, name)
	if err != nil {
		writeError(w, err)
		return
	}

	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		// noVNC/browser clients connect cross-origin from local pages and request "binary".
		InsecureSkipVerify: true,
		Subprotocols:       []string{"binary"},
	})
	if err != nil {
		return
	}
	s.activeConns.Add(1)
	defer func() {
		s.activeConns.Add(-1)
		s.touch()
		_ = c.Close(websocket.StatusInternalError, "proxy closed")
	}()

	vncConn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 10*time.Second)
	if err != nil {
		_ = c.Close(websocket.StatusInternalError, "vnc unreachable")
		return
	}
	defer vncConn.Close()

	// NetConn turns the WS into a net.Conn — a plain bidirectional pipe.
	wsConn := websocket.NetConn(r.Context(), c, websocket.MessageBinary)
	defer wsConn.Close()

	errCh := make(chan error, 2)
	go func() {
		_, err := io.Copy(vncConn, wsConn)
		errCh <- err
	}()
	go func() {
		_, err := io.Copy(wsConn, vncConn)
		errCh <- err
	}()
	<-errCh
	_ = c.Close(websocket.StatusNormalClosure, "done")
}

// displayEndpoint resolves an instance's VNC host/port from state.
func (s *Server) displayEndpoint(root, name string) (string, int, error) {
	metadata, err := project.LoadMetadata(root)
	if err != nil {
		return "", 0, err
	}
	yeastHome := s.opts.YeastHome
	if yeastHome == "" {
		yeastHome, err = project.DefaultYeastHome()
		if err != nil {
			return "", 0, err
		}
	}
	paths, err := project.NewPaths(yeastHome, metadata)
	if err != nil {
		return "", 0, err
	}
	currentState, err := state.Load(paths.StateFile, metadata.ID)
	if err != nil {
		return "", 0, err
	}
	inst, ok := currentState.Instances[name]
	if !ok {
		return "", 0, app.WrapError(app.ErrorCodeNotFound, fmt.Sprintf("instance %q not found", name), nil)
	}
	if inst.Display == nil || inst.Display.Port <= 0 {
		return "", 0, app.WrapError(app.ErrorCodePrecondition, fmt.Sprintf("instance %q has no display configured", name), nil)
	}
	host := inst.Display.Host
	if host == "" {
		host = "127.0.0.1"
	}
	return host, inst.Display.Port, nil
}

// displayEndpointForTest lets tests stub resolution.
