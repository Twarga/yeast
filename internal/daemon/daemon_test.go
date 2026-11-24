package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/Twarga/yeast/internal/app"
	"github.com/Twarga/yeast/internal/project"
	"github.com/Twarga/yeast/internal/state"
)

func newTestServer(t *testing.T) (*Server, *httptest.Server, string) {
	t.Helper()
	// Isolate the app.Service yeast-home resolution (~/.yeast) from the real
	// user home — handlers create a fresh app.Service internally.
	t.Setenv("HOME", t.TempDir())
	yeastHome := t.TempDir()
	srv, err := New(nil, Options{YeastHome: yeastHome})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	// init a project
	root := t.TempDir()
	svc := app.NewService()
	if _, err := svc.Init(app.InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	return srv, ts, root
}

func postJSON(t *testing.T, url string, body any) (int, map[string]any) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		reader = strings.NewReader(string(data))
	}
	resp, err := http.Post(url, "application/json", reader)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out map[string]any
	raw, _ := io.ReadAll(resp.Body)
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &out)
	}
	return resp.StatusCode, out
}

func getJSON(t *testing.T, url string) (int, map[string]any) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out map[string]any
	raw, _ := io.ReadAll(resp.Body)
	_ = json.Unmarshal(raw, &out)
	return resp.StatusCode, out
}

func registerProject(t *testing.T, ts *httptest.Server, root string) string {
	t.Helper()
	code, out := postJSON(t, ts.URL+"/v1/projects", map[string]any{"root": root})
	if code != 200 {
		t.Fatalf("register = %d %v", code, out)
	}
	data := out["data"].(map[string]any)
	return data["project_id"].(string)
}

func TestRegisterResolveList(t *testing.T) {
	srv, ts, root := newTestServer(t)
	id := registerProject(t, ts, root)
	if id == "" {
		t.Fatal("empty project id")
	}
	got, ok := srv.reg.Resolve(id)
	if !ok || got != root {
		t.Fatalf("resolve = %q %v", got, ok)
	}
	code, out := getJSON(t, ts.URL+"/v1/projects")
	if code != 200 {
		t.Fatalf("list = %d %v", code, out)
	}
	entries := out["data"].([]any)
	if len(entries) != 1 {
		t.Fatalf("entries = %v", entries)
	}
	// registry persisted
	reg2, err := NewRegistry(srv.opts.YeastHome)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := reg2.Resolve(id); !ok {
		t.Fatal("registry not persisted")
	}
}

func TestStatusRouteReturnsEmptyProject(t *testing.T) {
	_, ts, root := newTestServer(t)
	id := registerProject(t, ts, root)
	code, out := getJSON(t, ts.URL+"/v1/projects/"+id+"/status")
	if code != 200 {
		t.Fatalf("status = %d %v", code, out)
	}
	if out["ok"] != true {
		t.Fatalf("status resp = %v", out)
	}
}

func TestUnknownProject404(t *testing.T) {
	_, ts, _ := newTestServer(t)
	code, out := getJSON(t, ts.URL+"/v1/projects/nope/status")
	if code != http.StatusNotFound {
		t.Fatalf("code = %d %v", code, out)
	}
	errObj := out["error"].(map[string]any)
	if errObj["code"] != "not_found" {
		t.Fatalf("error = %v", errObj)
	}
}

func TestEventsHubStreamsBufferedAndLive(t *testing.T) {
	srv, ts, root := newTestServer(t)
	id := registerProject(t, ts, root)

	// publish one buffered event before connecting
	srv.hub(id).Publish(app.Event{Type: "progress", Name: "up.started", Command: "up", ProjectID: id})

	req, _ := http.NewRequest("GET", ts.URL+"/v1/projects/"+id+"/events", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("content-type = %s", ct)
	}
	reader := bufio.NewReader(resp.Body)
	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(line, "data: ") || !strings.Contains(line, "up.started") {
		t.Fatalf("first SSE line = %q", line)
	}
	// live event arrives too
	srv.hub(id).Publish(app.Event{Type: "progress", Name: "up.done", Command: "up", ProjectID: id})
	// skip blank line then read data line
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(line, "up.done") {
			return
		}
	}
	t.Fatal("live event not received")
}

func TestVNCProxyPipesToDisplayPort(t *testing.T) {
	srv, ts, root := newTestServer(t)
	id := registerProject(t, ts, root)

	// fake VNC server: echoes whatever it receives
	echoLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer echoLn.Close()
	port := echoLn.Addr().(*net.TCPAddr).Port
	go func() {
		for {
			c, err := echoLn.Accept()
			if err != nil {
				return
			}
			go func() { _, _ = io.Copy(c, c) }()
		}
	}()

	// point the instance's display at the fake VNC server in state
	writeDisplayState(t, srv, root, "web", port)

	ctx := context.Background()
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") +
		fmt.Sprintf("/v1/projects/%s/instances/web/vnc", id)
	conn, resp, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("ws dial: %v (resp %v)", err, resp)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	payload := []byte("RFB 003.008\n")
	if err := conn.Write(ctx, websocket.MessageBinary, payload); err != nil {
		t.Fatal(err)
	}
	_, got, err := conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(payload) {
		t.Fatalf("echo = %q", got)
	}
}

func TestVNCProxyNoDisplayPrecondition(t *testing.T) {
	srv, ts, root := newTestServer(t)
	id := registerProject(t, ts, root)
	writeDisplayState(t, srv, root, "web", 0)
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") +
		fmt.Sprintf("/v1/projects/%s/instances/web/vnc", id)
	_, resp, err := websocket.Dial(context.Background(), wsURL, nil)
	if err == nil {
		t.Fatal("expected dial failure")
	}
	if resp != nil && resp.StatusCode != http.StatusPreconditionFailed {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}

func TestBearerMiddlewareRejectsBadToken(t *testing.T) {
	srv, _, _ := newTestServer(t)
	srv.opts.Token = "secret"
	handler := srv.bearerMiddleware(srv.Handler())

	req := httptest.NewRequest("GET", "/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("no-token status = %d", rec.Code)
	}

	req = httptest.NewRequest("GET", "/healthz", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("good-token status = %d", rec.Code)
	}
}

func TestConcurrentProjectWritesShareLock(t *testing.T) {
	_, ts, root := newTestServer(t)
	id := registerProject(t, ts, root)
	svc := app.NewService()

	// Hammer status via HTTP + direct service calls — the shared state lock
	// must serialize without corruption or deadlock.
	done := make(chan error, 20)
	for i := 0; i < 10; i++ {
		go func() {
			code, out := getJSON(t, ts.URL+"/v1/projects/"+id+"/status")
			if code != 200 || out["ok"] != true {
				done <- fmt.Errorf("http status failed: %d %v", code, out)
				return
			}
			done <- nil
		}()
		go func() {
			_, err := svc.Status(context.Background(), app.StatusOptions{ProjectRoot: root})
			done <- err
		}()
	}
	for i := 0; i < 20; i++ {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
}

// writeDisplayState seeds instance "web" with a display port in the daemon's
// YeastHome state dir.
func writeDisplayState(t *testing.T, srv *Server, root, name string, port int) {
	t.Helper()
	metadata, err := project.LoadMetadata(root)
	if err != nil {
		t.Fatal(err)
	}
	paths, err := project.NewPaths(srv.opts.YeastHome, metadata)
	if err != nil {
		t.Fatal(err)
	}
	st := state.New(metadata.ID)
	inst := state.InstanceState{Status: "running", PID: 4242, RuntimeDir: "/tmp/rt"}
	if port > 0 {
		inst.Display = &state.DisplayState{Type: "vnc", Host: "127.0.0.1", Port: port}
	}
	st.Instances[name] = inst
	if err := state.Save(paths.StateFile, st); err != nil {
		t.Fatal(err)
	}
}
