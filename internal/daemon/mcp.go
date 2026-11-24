package daemon

import (
	"fmt"
	"io"
	"net/http"

	"github.com/Twarga/yeast/internal/app"
	"github.com/Twarga/yeast/internal/mcp"
)

// handleMCP is the streamable-HTTP MCP endpoint: POST a JSON-RPC message,
// get a JSON-RPC response. The project is chosen by ?project=<id> (daemon
// registry) or, when exactly one project is registered, implicitly.
func (s *Server) handleMCP(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("project")
	var root string
	if id != "" {
		resolved, ok := s.reg.Resolve(id)
		if !ok {
			writeError(w, app.WrapError(app.ErrorCodeNotFound, fmt.Sprintf("project %q not registered", id), nil))
			return
		}
		root = resolved
	} else {
		projects := s.reg.List()
		if len(projects) != 1 {
			writeError(w, app.WrapError(app.ErrorCodeInvalidArgument,
				fmt.Sprintf("?project=<id> required (%d projects registered)", len(projects)), nil))
			return
		}
		for _, v := range projects {
			root = v
		}
	}

	srvAny, _ := s.mcpServers.LoadOrStore(root, s.mustMCPServer(root))
	srv := srvAny.(*mcp.Server)
	if srv == nil {
		s.mcpServers.Delete(root)
		writeError(w, app.WrapError(app.ErrorCodePrecondition, fmt.Sprintf("project at %s is not a Yeast project", root), nil))
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, app.WrapError(app.ErrorCodeInvalidArgument, err.Error(), err))
		return
	}
	resp := srv.Handle(r.Context(), body)
	if resp == nil {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(resp)
}

func (s *Server) mustMCPServer(root string) *mcp.Server {
	srv, err := mcp.NewServer(s.service, root)
	if err != nil {
		return nil
	}
	return srv
}
