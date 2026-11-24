package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppendAndRead(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".yeast"), 0o700); err != nil {
		t.Fatal(err)
	}
	Append(root, Entry{Actor: "cli", Action: "up", DurationMs: 12})
	Append(root, Entry{Actor: "mcp:test", Action: "exec", Args: Redact(map[string]any{
		"command": "ls", "token": "sekret", "instance": "vm1",
	}), SnapshotIDs: []string{"snap-1"}})

	data, err := os.ReadFile(filepath.Join(root, ".yeast", "audit.jsonl"))
	if err != nil {
		t.Fatalf("read audit file: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	var e2 Entry
	if err := json.Unmarshal([]byte(lines[1]), &e2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if e2.Actor != "mcp:test" || e2.Args["token"] != "[redacted]" || e2.Args["command"] != "ls" {
		t.Fatalf("bad entry: %+v", e2)
	}
	if len(e2.SnapshotIDs) != 1 || e2.SnapshotIDs[0] != "snap-1" {
		t.Fatalf("snapshot ids lost: %+v", e2.SnapshotIDs)
	}

	entries, err := Read(root)
	if err != nil || len(entries) != 2 {
		t.Fatalf("Read: %v %d", err, len(entries))
	}
	if entries[0].Time == "" {
		t.Fatal("timestamp must be auto-stamped")
	}
	// file perms
	fi, _ := os.Stat(filepath.Join(root, ".yeast", "audit.jsonl"))
	if fi.Mode().Perm() != 0o600 {
		t.Fatalf("audit file must be 0600, got %o", fi.Mode().Perm())
	}
}

func TestReadMissingIsEmpty(t *testing.T) {
	entries, err := Read(t.TempDir())
	if err != nil || len(entries) != 0 {
		t.Fatalf("expected empty, got %v %v", entries, err)
	}
}
