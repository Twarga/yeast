package egress

import (
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProxyAllowAndDeny(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "network.log")
	p, err := NewProxy([]string{"allowed.example.com"}, logPath)
	if err != nil {
		t.Fatalf("NewProxy: %v", err)
	}
	defer p.Close()

	// Allowed: forwards (upstream fails in test env → non-403), logs ALLOW.
	recAllow := httptest.NewRecorder()
	p.ServeHTTP(recAllow, httptest.NewRequest("GET", "http://allowed.example.com/x", nil))
	if recAllow.Code == http.StatusForbidden {
		t.Fatalf("allowed domain got 403")
	}

	// Denied: 403 + DENY log.
	recDeny := httptest.NewRecorder()
	p.ServeHTTP(recDeny, httptest.NewRequest("GET", "http://evil.example.com/x", nil))
	if recDeny.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for denied domain, got %d", recDeny.Code)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	log := string(data)
	if !strings.Contains(log, "DENY") || !strings.Contains(log, "evil.example.com") {
		t.Fatalf("expected DENY entry for evil.example.com, log: %q", log)
	}
	if !strings.Contains(log, "ALLOW") || !strings.Contains(log, "allowed.example.com") {
		t.Fatalf("expected ALLOW entry for allowed.example.com, log: %q", log)
	}
}

func TestProxySubdomainSuffixMatch(t *testing.T) {
	p, err := NewProxy([]string{"example.com"}, "")
	if err != nil {
		t.Fatalf("NewProxy: %v", err)
	}
	defer p.Close()
	if !p.allowed("api.example.com:443") {
		t.Fatal("expected subdomain api.example.com to be allowed")
	}
	if p.allowed("notexample.com") {
		t.Fatal("notexample.com must not match example.com suffix")
	}
}

func TestProxyConnectDeny(t *testing.T) {
	p, err := NewProxy([]string{"ok.example.com"}, "")
	if err != nil {
		t.Fatalf("NewProxy: %v", err)
	}
	defer p.Close()
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, httptest.NewRequest(http.MethodConnect, "https://blocked.example.com:443", nil))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("CONNECT to unlisted host must be 403, got %d", rec.Code)
	}
}

func TestEnsurePort(t *testing.T) {
	if got := ensurePort("example.com", "443"); got != net.JoinHostPort("example.com", "443") {
		t.Fatalf("ensurePort: %q", got)
	}
	if got := ensurePort("example.com:8443", "443"); got != "example.com:8443" {
		t.Fatalf("ensurePort keep: %q", got)
	}
}
