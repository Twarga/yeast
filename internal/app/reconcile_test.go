package app

import (
	"context"
	"path/filepath"
	"testing"
	"time"
	"yeast/internal/project"
	rtm "yeast/internal/runtime"
	"yeast/internal/state"
)

func TestStatusReconcilesMissingPIDByRuntimeDir(t *testing.T) {
	root := t.TempDir()
	yeastHome := filepath.Join(root, "yeast-home")
	service := NewService()
	service.resolveYeastHome = func() (string, error) { return yeastHome, nil }
	service.runtime = &fakeReconcileRuntime{
		states: map[int]rtm.ProcessState{},
		found:  []rtm.CleanupResult{{Name: "web", PID: 5151}},
	}

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
	current.Instances["web"] = state.InstanceState{Status: "running", PID: 0, SSHPort: 2222, RuntimeDir: "/tmp/web"}
	if err := state.Save(paths.StateFile, current); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	result, err := service.Status(context.Background(), StatusOptions{ProjectRoot: root})
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if len(result.Instances) != 1 || result.Instances[0].PID != 5151 || result.Instances[0].Status != "running" {
		t.Fatalf("expected reconciled running pid, got %#v", result.Instances)
	}
	loaded, err := state.Load(paths.StateFile, metadata.ID)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if loaded.Instances["web"].PID != 5151 {
		t.Fatalf("expected saved pid 5151, got %#v", loaded.Instances["web"])
	}
}

type fakeReconcileRuntime struct {
	states map[int]rtm.ProcessState
	found  []rtm.CleanupResult
	err    error
}

func (f *fakeReconcileRuntime) PrepareDisk(ctx context.Context, plan rtm.MachinePlan) (rtm.DiskPlan, error) {
	return plan.Disk, nil
}

func (f *fakeReconcileRuntime) Start(ctx context.Context, plan rtm.MachinePlan) (rtm.RuntimeInstance, error) {
	return rtm.RuntimeInstance{}, nil
}

func (f *fakeReconcileRuntime) Stop(ctx context.Context, instance rtm.RuntimeInstance, timeout time.Duration) error {
	return nil
}

func (f *fakeReconcileRuntime) Inspect(ctx context.Context, instance rtm.RuntimeInstance) (rtm.ProcessInfo, error) {
	return rtm.ProcessInfo{PID: instance.PID, State: f.states[instance.PID]}, nil
}

func (f *fakeReconcileRuntime) CreateSnapshot(ctx context.Context, plan rtm.SnapshotPlan) error {
	return nil
}

func (f *fakeReconcileRuntime) RestoreSnapshot(ctx context.Context, plan rtm.SnapshotPlan) error {
	return nil
}

func (f *fakeReconcileRuntime) DeleteSnapshot(ctx context.Context, snapshotPath string) error {
	return nil
}

func (f *fakeReconcileRuntime) Destroy(ctx context.Context, instance rtm.RuntimeInstance) error {
	return nil
}

func (f *fakeReconcileRuntime) FindProcesses(ctx context.Context, targets []rtm.CleanupTarget) ([]rtm.CleanupResult, error) {
	return f.found, f.err
}
