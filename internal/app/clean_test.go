package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
	"yeast/internal/project"
	rtm "yeast/internal/runtime"
	"yeast/internal/state"
)

func TestCleanWorksWithoutStateFile(t *testing.T) {
	root := t.TempDir()
	yeastHome := filepath.Join(root, "yeast-home")
	service := NewService()
	service.resolveYeastHome = func() (string, error) { return yeastHome, nil }
	service.runtime = &fakeCleanRuntime{}

	if _, err := service.Init(InitOptions{ProjectRoot: root}); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	metadata, err := project.LoadMetadata(root)
	if err != nil {
		t.Fatalf("LoadMetadata returned error: %v", err)
	}
	paths, err := project.NewPaths(yeastHome, metadata)
	if err != nil {
		t.Fatalf("NewPaths returned error: %v", err)
	}
	if err := os.Remove(paths.StateFile); err != nil && !os.IsNotExist(err) {
		t.Fatalf("remove state file: %v", err)
	}

	result, err := service.Clean(context.Background(), CleanOptions{ProjectRoot: root})
	if err != nil {
		t.Fatalf("Clean returned error: %v", err)
	}
	if len(result.Instances) != 0 {
		t.Fatalf("expected no cleaned instances, got %#v", result.Instances)
	}
	loaded, err := state.Load(paths.StateFile, metadata.ID)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(loaded.Instances) != 0 {
		t.Fatalf("expected empty state after clean, got %#v", loaded.Instances)
	}
}

func TestCleanIgnoresBrokenConfigButUsesState(t *testing.T) {
	root := t.TempDir()
	yeastHome := filepath.Join(root, "yeast-home")
	service := NewService()
	service.resolveYeastHome = func() (string, error) { return yeastHome, nil }
	fakeRuntime := &fakeCleanRuntime{}
	service.runtime = fakeRuntime

	if _, err := service.Init(InitOptions{ProjectRoot: root}); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	metadata, err := project.LoadMetadata(root)
	if err != nil {
		t.Fatalf("LoadMetadata returned error: %v", err)
	}
	paths, err := project.NewPaths(yeastHome, metadata)
	if err != nil {
		t.Fatalf("NewPaths returned error: %v", err)
	}
	current := state.New(metadata.ID)
	current.Instances["web"] = state.InstanceState{Status: "running", PID: 4242, SSHPort: 2222, RuntimeDir: "/tmp/web"}
	if err := state.Save(paths.StateFile, current); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ConfigFileName), []byte("version: 1\ninstances:\n  - name:\n"), 0o644); err != nil {
		t.Fatalf("write broken config: %v", err)
	}

	result, err := service.Clean(context.Background(), CleanOptions{ProjectRoot: root})
	if err != nil {
		t.Fatalf("Clean returned error: %v", err)
	}
	if len(result.Instances) != 1 || result.Instances[0].Name != "web" {
		t.Fatalf("expected cleaned web instance, got %#v", result.Instances)
	}
	if len(fakeRuntime.destroyed) != 1 || fakeRuntime.destroyed[0].RuntimeDir != "/tmp/web" {
		t.Fatalf("expected runtime destroy by state-backed runtime dir, got %#v", fakeRuntime.destroyed)
	}
}

type fakeCleanRuntime struct {
	destroyed []rtm.RuntimeInstance
	cleaned   []rtm.CleanupResult
	cleanErr  error
}

func (f *fakeCleanRuntime) PrepareDisk(ctx context.Context, plan rtm.MachinePlan) (rtm.DiskPlan, error) {
	return plan.Disk, nil
}

func (f *fakeCleanRuntime) Start(ctx context.Context, plan rtm.MachinePlan) (rtm.RuntimeInstance, error) {
	return rtm.RuntimeInstance{}, nil
}

func (f *fakeCleanRuntime) Stop(ctx context.Context, instance rtm.RuntimeInstance, timeout time.Duration) error {
	return nil
}

func (f *fakeCleanRuntime) Inspect(ctx context.Context, instance rtm.RuntimeInstance) (rtm.ProcessInfo, error) {
	return rtm.ProcessInfo{}, nil
}

func (f *fakeCleanRuntime) CreateSnapshot(ctx context.Context, plan rtm.SnapshotPlan) error {
	return nil
}

func (f *fakeCleanRuntime) RestoreSnapshot(ctx context.Context, plan rtm.SnapshotPlan) error {
	return nil
}

func (f *fakeCleanRuntime) DeleteSnapshot(ctx context.Context, snapshotPath string) error {
	return nil
}

func (f *fakeCleanRuntime) Destroy(ctx context.Context, instance rtm.RuntimeInstance) error {
	f.destroyed = append(f.destroyed, instance)
	return nil
}

func (f *fakeCleanRuntime) CleanOrphans(ctx context.Context, targets []rtm.CleanupTarget, timeout time.Duration) ([]rtm.CleanupResult, error) {
	return f.cleaned, f.cleanErr
}
