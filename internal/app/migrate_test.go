package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Twarga/yeast/internal/project"
	"github.com/Twarga/yeast/internal/state"
)

func seedV1Project(t *testing.T) (root string, paths project.Paths, st state.State) {
	t.Helper()
	root = t.TempDir()
	id, err := project.GenerateID()
	if err != nil {
		t.Fatalf("generate id: %v", err)
	}
	metadata := project.NewMetadata(id, time.Now())
	if err := project.SaveMetadata(root, metadata); err != nil {
		t.Fatalf("save metadata: %v", err)
	}
	home := t.TempDir()
	yeastHome := filepath.Join(home, ".yeast")
	t.Setenv("HOME", home)
	p, err := project.NewPaths(yeastHome, metadata)
	if err != nil {
		t.Fatalf("new paths: %v", err)
	}
	// v1-era state: instance with a dead PID, a legacy snapshot without Kind,
	// and a stale .part artifact.
	st = state.New(metadata.ID)
	rtDir, err := p.InstanceDir("web")
	if err != nil {
		t.Fatalf("instance dir: %v", err)
	}
	if err := os.MkdirAll(rtDir, 0755); err != nil {
		t.Fatalf("runtime dir: %v", err)
	}
	stale := filepath.Join(rtDir, "download.img.part")
	if err := os.WriteFile(stale, []byte("partial"), 0644); err != nil {
		t.Fatalf("stale part: %v", err)
	}
	old := time.Now().Add(-2 * time.Hour)
	_ = os.Chtimes(stale, old, old)
	st.Instances["web"] = state.InstanceState{
		Status:     "running",
		PID:        999999, // dead
		RuntimeDir: rtDir,
		Snapshots: map[string]state.SnapshotState{
			"clean": {Name: "clean", CreatedAt: time.Now(), DiskPath: filepath.Join(rtDir, "snap-clean.qcow2")},
		},
	}
	if err := state.Save(p.StateFile, st); err != nil {
		t.Fatalf("save state: %v", err)
	}
	return root, p, st
}

func TestMigrateUpgradesV1State(t *testing.T) {
	root, paths, _ := seedV1Project(t)
	svc := NewService()

	res, err := svc.Migrate(context.Background(), MigrateOptions{Root: root})
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if res.ProjectID != paths.ProjectID {
		t.Fatalf("project_id = %q", res.ProjectID)
	}
	actions := map[string]bool{}
	for _, c := range res.Changes {
		actions[c.Action] = true
	}
	for _, want := range []string{"reconcile", "snapshot-kind", "create-snapshots-dir", "remove-stale"} {
		if !actions[want] {
			t.Errorf("missing change %q; got %v", want, res.Changes)
		}
	}

	// State on disk is now normalized.
	migrated, err := state.Load(paths.StateFile, paths.ProjectID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	inst := migrated.Instances["web"]
	if inst.PID != 0 || inst.Status == "running" {
		t.Errorf("dead PID not reconciled: pid=%d status=%q", inst.PID, inst.Status)
	}
	if inst.Snapshots["clean"].Kind != "copy" {
		t.Errorf("snapshot kind = %q, want copy", inst.Snapshots["clean"].Kind)
	}
	snapDir, _ := paths.SnapshotDir("web")
	if _, err := os.Stat(snapDir); err != nil {
		t.Errorf("snapshots dir not created: %v", err)
	}
	if _, err := os.Stat(filepath.Join(inst.RuntimeDir, "download.img.part")); !os.IsNotExist(err) {
		t.Errorf("stale .part not removed")
	}
}

func TestMigrateDryRunWritesNothing(t *testing.T) {
	root, paths, before := seedV1Project(t)
	svc := NewService()

	res, err := svc.Migrate(context.Background(), MigrateOptions{Root: root, DryRun: true})
	if err != nil {
		t.Fatalf("migrate --dry-run: %v", err)
	}
	if !res.DryRun {
		t.Fatal("dry_run flag not reflected")
	}
	if len(res.Changes) == 0 {
		t.Fatal("expected proposed changes")
	}

	// Nothing on disk changed: stale part still present, snapshots dir absent,
	// state file untouched.
	inst := before.Instances["web"]
	if _, err := os.Stat(filepath.Join(inst.RuntimeDir, "download.img.part")); err != nil {
		t.Errorf("dry-run removed .part file")
	}
	snapDir, _ := paths.SnapshotDir("web")
	if _, err := os.Stat(snapDir); err == nil {
		t.Errorf("dry-run created snapshots dir")
	}
	reloaded, _ := state.Load(paths.StateFile, paths.ProjectID)
	if reloaded.Instances["web"].Snapshots["clean"].Kind != "" {
		t.Errorf("dry-run modified state file")
	}
}

func TestMigrateRequiresProject(t *testing.T) {
	svc := NewService()
	_, err := svc.Migrate(context.Background(), MigrateOptions{Root: t.TempDir()})
	if err == nil {
		t.Fatal("expected error for non-project dir")
	}
	var appErr *AppError
	if !asAppError(err, &appErr) || appErr.Code != ErrorCodeNotFound {
		t.Fatalf("want typed not_found, got %v", err)
	}
}

// asAppError unwraps err to *AppError without importing errors helpers twice.
func asAppError(err error, target **AppError) bool {
	for err != nil {
		if e, ok := err.(*AppError); ok {
			*target = e
			return true
		}
		if u, ok := err.(interface{ Unwrap() error }); ok {
			err = u.Unwrap()
			continue
		}
		return false
	}
	return false
}

func TestMigrateResultJSONShape(t *testing.T) {
	res := MigrateResult{
		ProjectID: "p", Instances: 1,
		Changes: []MigrateChange{{Action: "reconcile"}},
	}
	raw, _ := json.Marshal(res)
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("migrate result not JSON: %v", err)
	}
	if decoded["project_id"] != "p" || decoded["changes"] == nil {
		t.Fatalf("bad shape: %s", raw)
	}
}
