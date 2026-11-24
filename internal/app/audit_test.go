package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Twarga/yeast/internal/audit"
)

func TestAuditCallWritesEntry(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".yeast"), 0o700); err != nil {
		t.Fatal(err)
	}
	s := NewService()
	ctx := WithActor(context.Background(), "api")
	errBoom := errors.New("boom")
	s.auditCall(ctx, root, "exec", map[string]any{
		"instance": "vm1", "command": "id", "token": "sekret",
	}, time.Now().Add(-5*time.Millisecond), errBoom, "snap-9")

	entries, err := audit.Read(root)
	if err != nil || len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %v %v", entries, err)
	}
	e := entries[0]
	if e.Actor != "api" || e.Action != "exec" || e.Error != "boom" {
		t.Fatalf("bad entry: %+v", e)
	}
	if e.Args["token"] != "[redacted]" {
		t.Fatalf("token not redacted: %+v", e.Args)
	}
	if len(e.SnapshotIDs) != 1 || e.SnapshotIDs[0] != "snap-9" {
		t.Fatalf("snapshot ids missing: %+v", e)
	}
	if e.DurationMs < 0 {
		t.Fatalf("duration negative: %d", e.DurationMs)
	}
}

func TestActorFromDefaults(t *testing.T) {
	if got := ActorFrom(context.Background()); got != "cli" {
		t.Fatalf("default actor must be cli, got %q", got)
	}
	if got := ActorFrom(WithActor(context.Background(), "mcp:cursor")); got != "mcp:cursor" {
		t.Fatalf("actor not propagated: %q", got)
	}
}
