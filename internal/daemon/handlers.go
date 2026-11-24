package daemon

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/Twarga/yeast/internal/app"
)

// ---------- helpers ----------------------------------------------------------

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeResult(w http.ResponseWriter, command string, data any) {
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"command": command,
		"data":    data,
	})
}

func errorStatus(code app.ErrorCode) int {
	switch code {
	case app.ErrorCodeInvalidArgument:
		return http.StatusBadRequest
	case app.ErrorCodeNotFound:
		return http.StatusNotFound
	case app.ErrorCodeConflict:
		return http.StatusConflict
	case app.ErrorCodePrecondition:
		return http.StatusPreconditionFailed
	case app.ErrorCodeTimeout:
		return http.StatusGatewayTimeout
	case app.ErrorCodeGuest, app.ErrorCodeRuntime, app.ErrorCodeProvisioning:
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
}

func writeError(w http.ResponseWriter, err error) {
	var appErr *app.AppError
	if !errors.As(err, &appErr) {
		appErr = app.NormalizeError(err)
	}
	status := errorStatus(appErr.Code)
	writeJSON(w, status, map[string]any{
		"ok": false,
		"error": map[string]any{
			"code":    appErr.Code,
			"message": appErr.Error(),
		},
	})
}

func decodeBody(r *http.Request, dst any) error {
	if r.Body == nil {
		return nil
	}
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("invalid JSON body: %w", err)
	}
	return nil
}

// projectRoot resolves the path param {id} through the registry.
func (s *Server) projectRoot(w http.ResponseWriter, r *http.Request) (string, bool) {
	id := r.PathValue("id")
	root, ok := s.reg.Resolve(id)
	if !ok {
		writeError(w, app.WrapError(app.ErrorCodeNotFound, fmt.Sprintf("project %q not registered with daemon", id), nil))
		return "", false
	}
	return root, true
}

// ---------- routes -----------------------------------------------------------

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "service": "yeastd", "version": s.service.Version()})
}

func (s *Server) handleRegisterProject(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Root string `json:"root"`
	}
	if err := decodeBody(r, &req); err != nil {
		writeError(w, app.WrapError(app.ErrorCodeInvalidArgument, err.Error(), err))
		return
	}
	if req.Root == "" {
		writeError(w, app.WrapError(app.ErrorCodeInvalidArgument, "root is required", nil))
		return
	}
	id, err := s.reg.Register(req.Root)
	if err != nil {
		writeError(w, err)
		return
	}
	writeResult(w, "register", map[string]any{"project_id": id, "root": req.Root})
}

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	projects := s.reg.List()
	entries := make([]map[string]string, 0, len(projects))
	for id, root := range projects {
		entries = append(entries, map[string]string{"project_id": id, "root": root})
	}
	writeResult(w, "projects", entries)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	root, ok := s.projectRoot(w, r)
	if !ok {
		return
	}
	result, err := s.service.Status(r.Context(), app.StatusOptions{ProjectRoot: root})
	if err != nil {
		writeError(w, err)
		return
	}
	writeResult(w, "status", result)
}

func (s *Server) handleUp(w http.ResponseWriter, r *http.Request) {
	root, ok := s.projectRoot(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	result, err := s.service.Up(r.Context(), app.UpOptions{
		ProjectRoot: root,
		Events:      s.hub(id).Sink(),
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeResult(w, "up", result)
}

func (s *Server) handleDown(w http.ResponseWriter, r *http.Request) {
	root, ok := s.projectRoot(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	result, err := s.service.Down(r.Context(), app.DownOptions{
		ProjectRoot: root,
		Events:      s.hub(id).Sink(),
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeResult(w, "down", result)
}

type execRequest struct {
	Instance  string            `json:"instance"`
	Command   []string          `json:"command"`
	TimeoutMs int64             `json:"timeout_ms"`
	Cwd       string            `json:"cwd"`
	Env       map[string]string `json:"env"`
	User      string            `json:"user"`
	StdinB64  string            `json:"stdin_b64"`
}

func (s *Server) handleExec(w http.ResponseWriter, r *http.Request) {
	root, ok := s.projectRoot(w, r)
	if !ok {
		return
	}
	var req execRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, app.WrapError(app.ErrorCodeInvalidArgument, err.Error(), err))
		return
	}
	var stdin []byte
	if req.StdinB64 != "" {
		data, err := base64.StdEncoding.DecodeString(req.StdinB64)
		if err != nil {
			writeError(w, app.WrapError(app.ErrorCodeInvalidArgument, "invalid stdin_b64", err))
			return
		}
		stdin = data
	}
	result, err := s.service.Exec(r.Context(), app.ExecOptions{
		GuestTargetOptions: app.GuestTargetOptions{ProjectRoot: root, Target: req.Instance},
		Command:            req.Command,
		Timeout:            time.Duration(req.TimeoutMs) * time.Millisecond,
		Cwd:                req.Cwd,
		Env:                req.Env,
		User:               req.User,
		Stdin:              stdin,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeResult(w, "exec", result)
}

type snapshotRequest struct {
	Instance    string `json:"instance"`
	Name        string `json:"name"`
	Description string `json:"description"`
	All         bool   `json:"all"`
}

func (s *Server) handleSnapshot(w http.ResponseWriter, r *http.Request) {
	root, ok := s.projectRoot(w, r)
	if !ok {
		return
	}
	var req snapshotRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, app.WrapError(app.ErrorCodeInvalidArgument, err.Error(), err))
		return
	}
	id := r.PathValue("id")
	if req.All {
		result, err := s.service.SnapshotAll(r.Context(), app.SnapshotAllOptions{
			ProjectRoot: root, Name: req.Name, Description: req.Description,
			Events: s.hub(id).Sink(),
		})
		if err != nil {
			writeError(w, err)
			return
		}
		writeResult(w, "snapshot", result)
		return
	}
	result, err := s.service.Snapshot(r.Context(), app.SnapshotOptions{
		ProjectRoot: root, Target: req.Instance, Name: req.Name, Description: req.Description,
		Events: s.hub(id).Sink(),
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeResult(w, "snapshot", result)
}

type restoreRequest struct {
	Instance string `json:"instance"`
	Name     string `json:"name"`
	All      bool   `json:"all"`
}

func (s *Server) handleRestore(w http.ResponseWriter, r *http.Request) {
	root, ok := s.projectRoot(w, r)
	if !ok {
		return
	}
	var req restoreRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, app.WrapError(app.ErrorCodeInvalidArgument, err.Error(), err))
		return
	}
	id := r.PathValue("id")
	if req.All {
		result, err := s.service.RestoreAll(r.Context(), app.RestoreAllOptions{
			ProjectRoot: root, Name: req.Name, Events: s.hub(id).Sink(),
		})
		if err != nil {
			writeError(w, err)
			return
		}
		writeResult(w, "restore", result)
		return
	}
	result, err := s.service.Restore(r.Context(), app.RestoreOptions{
		ProjectRoot: root, Target: req.Instance, Name: req.Name,
		Events: s.hub(id).Sink(),
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeResult(w, "restore", result)
}

type forkRequest struct {
	Instance string `json:"instance"`
	Count    int    `json:"count"`
	From     string `json:"from"`
	Boot     bool   `json:"boot"`
}

func (s *Server) handleFork(w http.ResponseWriter, r *http.Request) {
	root, ok := s.projectRoot(w, r)
	if !ok {
		return
	}
	var req forkRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, app.WrapError(app.ErrorCodeInvalidArgument, err.Error(), err))
		return
	}
	result, err := s.service.Fork(r.Context(), app.ForkOptions{
		ProjectRoot: root, Target: req.Instance, Count: req.Count,
		From: req.From, Boot: req.Boot,
		Events: s.hub(r.PathValue("id")).Sink(),
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeResult(w, "fork", result)
}

type screenshotRequest struct {
	Instance   string `json:"instance"`
	OutputPath string `json:"output_path"`
}

func (s *Server) handleScreenshot(w http.ResponseWriter, r *http.Request) {
	root, ok := s.projectRoot(w, r)
	if !ok {
		return
	}
	var req screenshotRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, app.WrapError(app.ErrorCodeInvalidArgument, err.Error(), err))
		return
	}
	result, err := s.service.Screenshot(r.Context(), app.ScreenshotOptions{
		ProjectRoot: root, Target: req.Instance, OutputPath: req.OutputPath,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeResult(w, "screenshot", result)
}

type inputRequest struct {
	Instance   string   `json:"instance"`
	Keys       []string `json:"keys"`
	Text       string   `json:"text"`
	HoldTimeMs int      `json:"hold_time_ms"`
}

func (s *Server) handleInput(w http.ResponseWriter, r *http.Request) {
	root, ok := s.projectRoot(w, r)
	if !ok {
		return
	}
	var req inputRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, app.WrapError(app.ErrorCodeInvalidArgument, err.Error(), err))
		return
	}
	result, err := s.service.Input(r.Context(), app.InputOptions{
		ProjectRoot: root, Target: req.Instance,
		Keys: req.Keys, Text: req.Text, HoldTimeMs: req.HoldTimeMs,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeResult(w, "input", result)
}

type cleanRequest struct {
	Snapshots bool `json:"snapshots"`
}

func (s *Server) handleClean(w http.ResponseWriter, r *http.Request) {
	root, ok := s.projectRoot(w, r)
	if !ok {
		return
	}
	var req cleanRequest
	_ = decodeBody(r, &req) // empty body is fine
	if req.Snapshots {
		result, err := s.service.CleanSnapshots(r.Context(), app.CleanOptions{ProjectRoot: root})
		if err != nil {
			writeError(w, err)
			return
		}
		writeResult(w, "clean", result)
		return
	}
	result, err := s.service.Clean(r.Context(), app.CleanOptions{ProjectRoot: root})
	if err != nil {
		writeError(w, err)
		return
	}
	writeResult(w, "clean", result)
}

type destroyRequest struct {
	KeepFiles bool   `json:"keep_files"`
	Forks     string `json:"forks"`
}

func (s *Server) handleDestroy(w http.ResponseWriter, r *http.Request) {
	root, ok := s.projectRoot(w, r)
	if !ok {
		return
	}
	var req destroyRequest
	_ = decodeBody(r, &req)
	if req.Forks != "" {
		result, err := s.service.DestroyForks(r.Context(), app.DestroyOptions{ProjectRoot: root}, req.Forks)
		if err != nil {
			writeError(w, err)
			return
		}
		writeResult(w, "destroy", result)
		return
	}
	result, err := s.service.Destroy(r.Context(), app.DestroyOptions{
		ProjectRoot: root, KeepFiles: req.KeepFiles,
		Events: s.hub(r.PathValue("id")).Sink(),
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeResult(w, "destroy", result)
}

func (s *Server) handleSnapshots(w http.ResponseWriter, r *http.Request) {
	root, ok := s.projectRoot(w, r)
	if !ok {
		return
	}
	result, err := s.service.Snapshots(r.Context(), app.SnapshotsOptions{
		ProjectRoot: root, Target: r.URL.Query().Get("instance"),
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeResult(w, "snapshots", result)
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	root, ok := s.projectRoot(w, r)
	if !ok {
		return
	}
	result, err := s.service.Logs(r.Context(), app.LogsOptions{
		GuestTargetOptions: app.GuestTargetOptions{
			ProjectRoot: root, Target: r.URL.Query().Get("instance"),
		},
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeResult(w, "logs", result)
}

// handleEvents streams the project's event hub as Server-Sent Events.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, ok := s.reg.Resolve(id); !ok {
		writeError(w, app.WrapError(app.ErrorCodeNotFound, fmt.Sprintf("project %q not registered", id), nil))
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, app.WrapError(app.ErrorCodeInternal, "streaming unsupported", nil))
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	buffered, live, cancel := s.hub(id).Subscribe()
	defer cancel()

	send := func(e app.Event) error {
		data, err := json.Marshal(e)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
			return err
		}
		flusher.Flush()
		return nil
	}
	for _, e := range buffered {
		if err := send(e); err != nil {
			return
		}
	}
	for {
		select {
		case <-r.Context().Done():
			return
		case e, ok := <-live:
			if !ok {
				return
			}
			if err := send(e); err != nil {
				return
			}
		}
	}
}

type exportRequest struct {
	Instance string `json:"instance"`
	Format   string `json:"format"`
	Output   string `json:"output"`
	Sysprep  bool   `json:"sysprep"`
	SkelUser string `json:"skel_user"`
	Builder  string `json:"builder"`
}

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	root, ok := s.projectRoot(w, r)
	if !ok {
		return
	}
	var req exportRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, app.WrapError(app.ErrorCodeInvalidArgument, err.Error(), err))
		return
	}
	result, err := s.service.Export(r.Context(), app.ExportOptions{
		GuestTargetOptions: app.GuestTargetOptions{ProjectRoot: root, Target: req.Instance},
		Format:             req.Format,
		Output:             req.Output,
		Sysprep:            req.Sysprep,
		SkelUser:           req.SkelUser,
		Builder:            req.Builder,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeResult(w, "export", result)
}
