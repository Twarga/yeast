// Package yeast is the public Go SDK for the Yeast local daemon (yeastd).
// It talks to the daemon over its user-private unix socket and returns the
// same typed results the CLI uses.
//
//	c, _ := yeast.Connect("")
//	id, _ := c.RegisterProject(ctx, "/path/to/project")
//	status, _ := c.Status(ctx, id)
package yeast

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/Twarga/yeast/internal/app"
)

// Client talks to a running yeastd.
type Client struct {
	http   *http.Client
	socket string
	token  string
	base   string
}

// Connect dials the daemon. socketPath empty uses the default
// ($XDG_RUNTIME_DIR/yeast/yeastd.sock). For TCP listeners pass
// ConnectTCP(addr, token).
func Connect(socketPath string) (*Client, error) {
	if socketPath == "" {
		socketPath = defaultSocketPath()
	}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, "unix", socketPath)
		},
	}
	return &Client{
		http:   &http.Client{Transport: transport, Timeout: 0},
		socket: socketPath,
		base:   "http://yeastd",
	}, nil
}

// ConnectTCP connects to a --tcp listener (bearer token required).
func ConnectTCP(addr, token string) (*Client, error) {
	return &Client{
		http:  &http.Client{Timeout: 0},
		base:  "http://" + addr,
		token: token,
	}, nil
}

func defaultSocketPath() string {
	if dir := os.Getenv("XDG_RUNTIME_DIR"); dir != "" {
		return filepath.Join(dir, "yeast", "yeastd.sock")
	}
	return filepath.Join(os.TempDir(), fmt.Sprintf("yeast-%d", os.Getuid()), "yeastd.sock")
}

// ---------- plumbing ---------------------------------------------------------

type envelope struct {
	OK      bool            `json:"ok"`
	Command string          `json:"command"`
	Data    json.RawMessage `json:"data"`
	Error   *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// Error is an API error returned by the daemon.
type Error struct {
	Code    string
	Message string
	Status  int
}

func (e *Error) Error() string { return fmt.Sprintf("%s: %s", e.Code, e.Message) }

func (c *Client) call(ctx context.Context, method, path string, body any, dst any) error {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var env envelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	if !env.OK || resp.StatusCode >= 400 {
		e := &Error{Status: resp.StatusCode, Code: "unknown", Message: "request failed"}
		if env.Error != nil {
			e.Code, e.Message = env.Error.Code, env.Error.Message
		}
		return e
	}
	if dst != nil && len(env.Data) > 0 {
		return json.Unmarshal(env.Data, dst)
	}
	return nil
}

// ---------- projects ---------------------------------------------------------

// RegisterProject registers a project root with the daemon and returns its
// project ID (used by every other method).
func (c *Client) RegisterProject(ctx context.Context, root string) (string, error) {
	var out struct {
		ProjectID string `json:"project_id"`
	}
	if err := c.call(ctx, http.MethodPost, "/v1/projects", map[string]any{"root": root}, &out); err != nil {
		return "", err
	}
	return out.ProjectID, nil
}

// Projects lists registered projects (id → root).
func (c *Client) Projects(ctx context.Context) (map[string]string, error) {
	var entries []struct {
		ProjectID string `json:"project_id"`
		Root      string `json:"root"`
	}
	if err := c.call(ctx, http.MethodGet, "/v1/projects", nil, &entries); err != nil {
		return nil, err
	}
	out := make(map[string]string, len(entries))
	for _, e := range entries {
		out[e.ProjectID] = e.Root
	}
	return out, nil
}

// ---------- instance lifecycle -----------------------------------------------

func (c *Client) Status(ctx context.Context, projectID string) (app.StatusResult, error) {
	var out app.StatusResult
	err := c.call(ctx, http.MethodGet, "/v1/projects/"+projectID+"/status", nil, &out)
	return out, err
}

func (c *Client) Up(ctx context.Context, projectID string) (app.UpResult, error) {
	var out app.UpResult
	err := c.call(ctx, http.MethodPost, "/v1/projects/"+projectID+"/up", map[string]any{}, &out)
	return out, err
}

func (c *Client) Down(ctx context.Context, projectID string) (app.DownResult, error) {
	var out app.DownResult
	err := c.call(ctx, http.MethodPost, "/v1/projects/"+projectID+"/down", map[string]any{}, &out)
	return out, err
}

func (c *Client) Destroy(ctx context.Context, projectID string) (app.DestroyResult, error) {
	var out app.DestroyResult
	err := c.call(ctx, http.MethodPost, "/v1/projects/"+projectID+"/destroy", map[string]any{}, &out)
	return out, err
}

// ---------- guest control ----------------------------------------------------

// ExecRequest mirrors the daemon's exec endpoint.
type ExecRequest struct {
	Instance  string            `json:"instance,omitempty"`
	Command   []string          `json:"command"`
	TimeoutMs int64             `json:"timeout_ms,omitempty"`
	Cwd       string            `json:"cwd,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
	User      string            `json:"user,omitempty"`
	Stdin     []byte            `json:"-"`
}

func (c *Client) Exec(ctx context.Context, projectID string, req ExecRequest) (app.ExecResult, error) {
	var out app.ExecResult
	body := map[string]any{
		"command":    req.Command,
		"timeout_ms": req.TimeoutMs,
	}
	if req.Instance != "" {
		body["instance"] = req.Instance
	}
	if req.Cwd != "" {
		body["cwd"] = req.Cwd
	}
	if len(req.Env) > 0 {
		body["env"] = req.Env
	}
	if req.User != "" {
		body["user"] = req.User
	}
	if len(req.Stdin) > 0 {
		body["stdin_b64"] = base64.StdEncoding.EncodeToString(req.Stdin)
	}
	err := c.call(ctx, http.MethodPost, "/v1/projects/"+projectID+"/exec", body, &out)
	return out, err
}

func (c *Client) Snapshot(ctx context.Context, projectID, instance, name string) (app.SnapshotResult, error) {
	var out app.SnapshotResult
	err := c.call(ctx, http.MethodPost, "/v1/projects/"+projectID+"/snapshot",
		map[string]any{"instance": instance, "name": name}, &out)
	return out, err
}

func (c *Client) SnapshotAll(ctx context.Context, projectID, name string) (app.SnapshotAllResult, error) {
	var out app.SnapshotAllResult
	err := c.call(ctx, http.MethodPost, "/v1/projects/"+projectID+"/snapshot",
		map[string]any{"name": name, "all": true}, &out)
	return out, err
}

func (c *Client) Restore(ctx context.Context, projectID, instance, name string) (app.RestoreResult, error) {
	var out app.RestoreResult
	err := c.call(ctx, http.MethodPost, "/v1/projects/"+projectID+"/restore",
		map[string]any{"instance": instance, "name": name}, &out)
	return out, err
}

func (c *Client) RestoreAll(ctx context.Context, projectID, name string) (app.RestoreAllResult, error) {
	var out app.RestoreAllResult
	err := c.call(ctx, http.MethodPost, "/v1/projects/"+projectID+"/restore",
		map[string]any{"name": name, "all": true}, &out)
	return out, err
}

func (c *Client) Fork(ctx context.Context, projectID, instance string, count int, from string, boot bool) (app.ForkResult, error) {
	var out app.ForkResult
	err := c.call(ctx, http.MethodPost, "/v1/projects/"+projectID+"/fork",
		map[string]any{"instance": instance, "count": count, "from": from, "boot": boot}, &out)
	return out, err
}

func (c *Client) Screenshot(ctx context.Context, projectID, instance string) (app.ScreenshotResult, error) {
	var out app.ScreenshotResult
	err := c.call(ctx, http.MethodPost, "/v1/projects/"+projectID+"/screenshot",
		map[string]any{"instance": instance}, &out)
	return out, err
}

func (c *Client) InputText(ctx context.Context, projectID, instance, text string) (app.InputResult, error) {
	var out app.InputResult
	err := c.call(ctx, http.MethodPost, "/v1/projects/"+projectID+"/input",
		map[string]any{"instance": instance, "text": text}, &out)
	return out, err
}

func (c *Client) InputKeys(ctx context.Context, projectID, instance string, keys []string) (app.InputResult, error) {
	var out app.InputResult
	err := c.call(ctx, http.MethodPost, "/v1/projects/"+projectID+"/input",
		map[string]any{"instance": instance, "keys": keys}, &out)
	return out, err
}

func (c *Client) Snapshots(ctx context.Context, projectID, instance string) (app.SnapshotsResult, error) {
	var out app.SnapshotsResult
	err := c.call(ctx, http.MethodGet,
		"/v1/projects/"+projectID+"/snapshots?instance="+instance, nil, &out)
	return out, err
}

// ---------- events -----------------------------------------------------------

// Events streams project events until ctx is cancelled. The returned channel
// delivers buffered history then live events.
func (c *Client) Events(ctx context.Context, projectID string) (<-chan app.Event, <-chan error) {
	events := make(chan app.Event, 64)
	errs := make(chan error, 1)
	go func() {
		defer close(events)
		defer close(errs)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet,
			c.base+"/v1/projects/"+projectID+"/events", nil)
		if err != nil {
			errs <- err
			return
		}
		if c.token != "" {
			req.Header.Set("Authorization", "Bearer "+c.token)
		}
		resp, err := c.http.Do(req)
		if err != nil {
			errs <- err
			return
		}
		defer resp.Body.Close()
		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			if len(line) < 6 || line[:6] != "data: " {
				continue
			}
			var e app.Event
			if err := json.Unmarshal([]byte(line[6:]), &e); err != nil {
				continue
			}
			select {
			case events <- e:
			case <-ctx.Done():
				return
			}
		}
		if err := scanner.Err(); err != nil && ctx.Err() == nil {
			errs <- err
		}
	}()
	return events, errs
}

// VNCWebSocketURL returns the ws:// URL for the built-in VNC proxy
// (unix-socket clients must dial the daemon's TCP listener instead).
func (c *Client) VNCWebSocketURL(projectID, instance string) string {
	base := c.base
	if c.socket != "" {
		base = "ws://localhost" // unix-socket transport can't WS-dial; use --tcp
	} else {
		base = "ws://" + c.base[len("http://"):]
	}
	return fmt.Sprintf("%s/v1/projects/%s/instances/%s/vnc", base, projectID, instance)
}
