// Package mcp implements a native Model Context Protocol server for Yeast:
// JSON-RPC 2.0 over stdio (`yeast mcp`) or streamable HTTP (`POST /mcp` on
// yeastd). Tools delegate directly to app.Service — no subprocesses.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/Twarga/yeast/internal/app"
)

const protocolVersion = "2024-11-05"

// Server is a tool-level MCP endpoint bound to one project root.
type Server struct {
	service *app.Service
	root    string
	policy  *Policy

	mu         sync.Mutex
	initDone   bool
	sessions   map[string]bool
	clientName string
}

// NewServer builds an MCP server for a project root.
func NewServer(service *app.Service, projectRoot string) (*Server, error) {
	if service == nil {
		service = app.NewService()
	}
	abs, err := filepath.Abs(projectRoot)
	if err != nil {
		return nil, err
	}
	policy, err := LoadPolicy(abs)
	if err != nil {
		return nil, fmt.Errorf("agent policy: %w", err)
	}
	return &Server{
		service:  service,
		root:     abs,
		policy:   policy,
		sessions: map[string]bool{},
	}, nil
}

// ---------- JSON-RPC plumbing ------------------------------------------------

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func errResp(id json.RawMessage, code int, msg string) rpcResponse {
	return rpcResponse{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: msg}}
}

func okResp(id json.RawMessage, result any) rpcResponse {
	return rpcResponse{JSONRPC: "2.0", ID: id, Result: result}
}

// Handle processes one JSON-RPC message and returns a response (nil for
// notifications). It is safe for concurrent use.
func (s *Server) Handle(ctx context.Context, raw []byte) []byte {
	var req rpcRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		out, _ := json.Marshal(errResp(nil, -32700, "parse error"))
		return out
	}
	resp, isNotification := s.dispatch(ctx, req)
	if isNotification {
		return nil
	}
	out, _ := json.Marshal(resp)
	return out
}

func (s *Server) dispatch(ctx context.Context, req rpcRequest) (rpcResponse, bool) {
	switch req.Method {
	case "initialize":
		var params struct {
			ClientInfo struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"clientInfo"`
		}
		_ = json.Unmarshal(req.Params, &params)
		s.mu.Lock()
		s.initDone = true
		if params.ClientInfo.Name != "" {
			s.sessions[params.ClientInfo.Name] = true
			s.clientName = params.ClientInfo.Name
		}
		s.mu.Unlock()
		return okResp(req.ID, map[string]any{
			"protocolVersion": protocolVersion,
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "yeast", "version": s.service.Version()},
		}), false

	case "notifications/initialized", "notifications/cancelled":
		return rpcResponse{}, true

	case "ping":
		return okResp(req.ID, map[string]any{}), false

	case "tools/list":
		return okResp(req.ID, map[string]any{"tools": s.toolList()}), false

	case "tools/call":
		var params struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return errResp(req.ID, -32602, "invalid tools/call params"), false
		}
		return s.callTool(ctx, req.ID, params.Name, params.Arguments), false

	default:
		if len(req.ID) == 0 {
			return rpcResponse{}, true // unknown notification
		}
		return errResp(req.ID, -32601, "method not found"), false
	}
}

// ---------- tool results -----------------------------------------------------

type contentBlock struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	Data     string `json:"data,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
}

func textResult(text string) map[string]any {
	return map[string]any{
		"content": []contentBlock{{Type: "text", Text: text}},
	}
}

func imageResult(b64, mime string) map[string]any {
	return map[string]any{
		"content": []contentBlock{{Type: "image", Data: b64, MimeType: mime}},
	}
}

func errorResult(err error) map[string]any {
	return map[string]any{
		"content": []contentBlock{{Type: "text", Text: err.Error()}},
		"isError": true,
	}
}

// callTool enforces the server-side policy then dispatches.
func (s *Server) callTool(ctx context.Context, id json.RawMessage, name string, args json.RawMessage) rpcResponse {
	actor := "mcp"
	s.mu.Lock()
	if s.clientName != "" {
		actor = "mcp:" + s.clientName
	}
	s.mu.Unlock()
	ctx = app.WithActor(ctx, actor)
	if !s.policy.Allows(name) {
		return okResp(id, errorResult(fmt.Errorf("tool %q denied by .yeast/agent-policy.yaml", name)))
	}
	result := s.invokeTool(ctx, name, args)
	return okResp(id, result)
}

// Stdio loop: newline-delimited JSON-RPC frames on stdin/stdout.
func (s *Server) ServeStdio(ctx context.Context, stdin *os.File, stdout *os.File) error {
	scanner := newLineScanner(stdin)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		resp := s.Handle(ctx, line)
		if resp != nil {
			if _, err := stdout.Write(append(resp, '\n')); err != nil {
				return err
			}
		}
	}
	return scanner.Err()
}
