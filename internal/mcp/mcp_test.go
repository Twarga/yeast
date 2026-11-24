package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Twarga/yeast/internal/app"
)

func newTestServer(t *testing.T) (*Server, string) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()
	svc := app.NewService()
	if _, err := svc.Init(app.InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	srv, err := NewServer(svc, root)
	if err != nil {
		t.Fatal(err)
	}
	return srv, root
}

func rpc(t *testing.T, srv *Server, method string, params any) map[string]any {
	t.Helper()
	var rawParams json.RawMessage
	if params != nil {
		rawParams, _ = json.Marshal(params)
	}
	req, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": method, "params": rawParams,
	})
	resp := srv.Handle(context.Background(), req)
	var out map[string]any
	if err := json.Unmarshal(resp, &out); err != nil {
		t.Fatalf("bad response: %v raw %s", err, resp)
	}
	return out
}

func TestInitializeReturnsCapabilities(t *testing.T) {
	srv, _ := newTestServer(t)
	out := rpc(t, srv, "initialize", map[string]any{
		"protocolVersion": "2024-11-05",
		"clientInfo":      map[string]any{"name": "test-client", "version": "1"},
	})
	result := out["result"].(map[string]any)
	if result["protocolVersion"] != "2024-11-05" {
		t.Fatalf("result = %v", result)
	}
	if result["serverInfo"].(map[string]any)["name"] != "yeast" {
		t.Fatalf("serverInfo = %v", result["serverInfo"])
	}
}

func TestToolsListContainsAllTools(t *testing.T) {
	srv, _ := newTestServer(t)
	out := rpc(t, srv, "tools/list", nil)
	tools := out["result"].(map[string]any)["tools"].([]any)
	if len(tools) != 18 {
		t.Fatalf("tools = %d", len(tools))
	}
	names := map[string]bool{}
	for _, tl := range tools {
		names[tl.(map[string]any)["name"].(string)] = true
	}
	for _, want := range []string{
		"vm_status", "vm_logs", "vm_ports", "vm_snapshots", "vm_read_file", "vm_screenshot",
		"vm_up", "vm_down", "vm_exec", "vm_write_file", "vm_copy_in", "vm_copy_out",
		"vm_input", "vm_snapshot", "vm_restore", "vm_fork", "vm_export", "vm_destroy",
	} {
		if !names[want] {
			t.Fatalf("missing tool %s", want)
		}
	}
	// destructive annotation on vm_destroy
	for _, tl := range tools {
		m := tl.(map[string]any)
		if m["name"] == "vm_destroy" {
			if m["annotations"].(map[string]any)["destructiveHint"] != true {
				t.Fatal("vm_destroy missing destructiveHint")
			}
		}
	}
}

func TestPolicyDenyBlocksTool(t *testing.T) {
	srv, root := newTestServer(t)
	policy := `
deny: [vm_destroy, vm_exec]
`
	if err := os.MkdirAll(filepath.Join(root, ".yeast"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".yeast", "agent-policy.yaml"), []byte(policy), 0644); err != nil {
		t.Fatal(err)
	}
	p, err := LoadPolicy(root)
	if err != nil {
		t.Fatal(err)
	}
	srv.policy = p

	out := rpc(t, srv, "tools/call", map[string]any{
		"name":      "vm_destroy",
		"arguments": map[string]any{"confirm": true},
	})
	content := out["result"].(map[string]any)["content"].([]any)
	if out["result"].(map[string]any)["isError"] != true {
		t.Fatalf("expected isError, got %v", out["result"])
	}
	if got := content[0].(map[string]any)["text"].(string); got == "" || !contains(got, "denied") {
		t.Fatalf("text = %q", got)
	}
}

func TestPolicyAllowListRestrictsTools(t *testing.T) {
	p := &Policy{Allow: []string{"vm_status"}}
	if !p.Allows("vm_status") || p.Allows("vm_exec") {
		t.Fatal("allow list not enforced")
	}
}

func TestPolicyClampsExecTimeout(t *testing.T) {
	p := &Policy{MaxExecTimeout: 30 * time.Second}
	if got := p.ClampExecTimeout(2 * time.Minute); got != 30*time.Second {
		t.Fatalf("clamp = %v", got)
	}
	if got := p.ClampExecTimeout(10 * time.Second); got != 10*time.Second {
		t.Fatalf("clamp = %v", got)
	}
}

func TestToolsCallStatusOnEmptyProject(t *testing.T) {
	srv, _ := newTestServer(t)
	out := rpc(t, srv, "tools/call", map[string]any{
		"name":      "vm_status",
		"arguments": map[string]any{},
	})
	result := out["result"].(map[string]any)
	if result["isError"] == true {
		t.Fatalf("result = %v", result)
	}
	content := result["content"].([]any)[0].(map[string]any)
	if content["type"] != "text" || content["text"] == "" {
		t.Fatalf("content = %v", content)
	}
}

func TestUnknownToolIsError(t *testing.T) {
	srv, _ := newTestServer(t)
	out := rpc(t, srv, "tools/call", map[string]any{
		"name":      "nope",
		"arguments": map[string]any{},
	})
	if out["result"].(map[string]any)["isError"] != true {
		t.Fatal("expected isError")
	}
}

func TestNotificationsReturnNil(t *testing.T) {
	srv, _ := newTestServer(t)
	req, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "method": "notifications/initialized",
	})
	if resp := srv.Handle(context.Background(), req); resp != nil {
		t.Fatalf("notification produced response: %s", resp)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		(func() bool {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		})())
}
