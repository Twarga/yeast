// Package audit writes Yeast's append-only action log at
// .yeast/audit.jsonl. Every mutating operation through the CLI, the
// daemon API, or the MCP server records actor, action, redacted args,
// duration, and involved snapshot IDs.
package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Entry is one audit record (one JSONL line).
type Entry struct {
	Time        string         `json:"time"`
	Actor       string         `json:"actor"` // "cli" | "api" | "mcp:<client>"
	Action      string         `json:"action"`
	Args        map[string]any `json:"args,omitempty"`
	DurationMs  int64          `json:"duration_ms"`
	SnapshotIDs []string       `json:"snapshot_ids,omitempty"`
	Error       string         `json:"error,omitempty"`
}

var mu sync.Mutex

// Append writes one entry to <projectRoot>/.yeast/audit.jsonl. Failures
// never break the caller — audit is best-effort by design.
func Append(projectRoot string, e Entry) {
	if e.Time == "" {
		e.Time = time.Now().UTC().Format(time.RFC3339)
	}
	if e.Actor == "" {
		e.Actor = "cli"
	}
	dir := filepath.Join(projectRoot, ".yeast")
	// Only audit where a project actually exists — a destroy that removed
	// .yeast must not resurrect it.
	if _, err := os.Stat(dir); err != nil {
		return
	}
	path := filepath.Join(dir, "audit.jsonl")
	mu.Lock()
	defer mu.Unlock()
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	data, err := json.Marshal(e)
	if err != nil {
		return
	}
	_, _ = f.Write(append(data, '\n'))
}

// sensitiveKeys are argument names whose values must never reach the log.
var sensitiveKeys = []string{"password", "token", "secret", "key", "stdin"}

// Redact returns a copy of args with sensitive values replaced.
func Redact(args map[string]any) map[string]any {
	if len(args) == 0 {
		return nil
	}
	out := make(map[string]any, len(args))
	for k, v := range args {
		lower := strings.ToLower(k)
		sensitive := false
		for _, s := range sensitiveKeys {
			if strings.Contains(lower, s) {
				sensitive = true
				break
			}
		}
		if sensitive {
			out[k] = "[redacted]"
		} else {
			out[k] = v
		}
	}
	return out
}

// Read returns all entries; missing file yields an empty slice.
func Read(projectRoot string) ([]Entry, error) {
	data, err := os.ReadFile(filepath.Join(projectRoot, ".yeast", "audit.jsonl"))
	if err != nil {
		if os.IsNotExist(err) {
			return []Entry{}, nil
		}
		return nil, fmt.Errorf("read audit log: %w", err)
	}
	var entries []Entry
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		var e Entry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue // skip corrupt lines rather than fail the whole log
		}
		entries = append(entries, e)
	}
	return entries, nil
}
