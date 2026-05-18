package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"yeast/internal/project"
	rtm "yeast/internal/runtime"
	"yeast/internal/state"
)

func TestSnapshotCreatesSnapshotAndPersistsMetadata(t *testing.T) {
	service, root, metadata, fakeRuntime := newSnapshotServiceWithStoppedInstance(t)

	result, err := service.Snapshot(context.Background(), SnapshotOptions{
		ProjectRoot: root,
		Target:      "web",
		Name:        "clean",
		Description: "Ready baseline",
	})
	if err != nil {
		t.Fatalf("Snapshot returned error: %v", err)
	}
	if result.ProjectID != metadata.ID {
		t.Fatalf("unexpected project id %q", result.ProjectID)
	}
	if result.Instance != "web" {
		t.Fatalf("unexpected snapshot instance %q", result.Instance)
	}
	if result.Snapshot.Name != "clean" {
		t.Fatalf("unexpected snapshot name %#v", result.Snapshot)
	}
	if result.Snapshot.Description != "Ready baseline" {
		t.Fatalf("unexpected snapshot description %#v", result.Snapshot)
	}
	if fakeRuntime.snapshotPlan.InstanceDiskPath == "" || fakeRuntime.snapshotPlan.SnapshotPath == "" {
		t.Fatalf("expected snapshot runtime plan to be populated, got %#v", fakeRuntime.snapshotPlan)
	}

	statePath := filepath.Join(root, "yeast-home", "projects", metadata.ID, "state.json")
	loaded, err := state.Load(statePath, metadata.ID)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	snapshot, ok := loaded.Instances["web"].Snapshots["clean"]
	if !ok {
		t.Fatalf("expected snapshot metadata to be persisted, got %#v", loaded.Instances["web"])
	}
	if snapshot.DiskPath != fakeRuntime.snapshotPlan.SnapshotPath {
		t.Fatalf("expected persisted disk path %q, got %q", fakeRuntime.snapshotPlan.SnapshotPath, snapshot.DiskPath)
	}
}

func TestSnapshotRequiresStoppedInstance(t *testing.T) {
	service, root, metadata, _ := newSnapshotServiceWithStoppedInstance(t)

	paths, err := project.NewPaths(filepath.Join(root, "yeast-home"), metadata)
	if err != nil {
		t.Fatalf("NewPaths returned error: %v", err)
	}
	current := state.New(metadata.ID)
	current.Instances["web"] = state.InstanceState{
		Status:     "running",
		PID:        os.Getpid(),
		RuntimeDir: filepath.Join(root, "yeast-home", "projects", metadata.ID, "instances", "web"),
	}
	if err := state.Save(paths.StateFile, current); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	_, err = service.Snapshot(context.Background(), SnapshotOptions{
		ProjectRoot: root,
		Target:      "web",
		Name:        "clean",
	})
	assertAppErrorCode(t, err, ErrorCodePrecondition)
}

func TestSnapshotRejectsDuplicateNames(t *testing.T) {
	service, root, metadata, _ := newSnapshotServiceWithStoppedInstance(t)

	paths, err := project.NewPaths(filepath.Join(root, "yeast-home"), metadata)
	if err != nil {
		t.Fatalf("NewPaths returned error: %v", err)
	}
	current := state.New(metadata.ID)
	runtimeDir := filepath.Join(root, "yeast-home", "projects", metadata.ID, "instances", "web")
	current.Instances["web"] = state.InstanceState{
		Status:     "stopped",
		RuntimeDir: runtimeDir,
		Snapshots: map[string]state.SnapshotState{
			"clean": {
				Name:      "clean",
				CreatedAt: time.Date(2026, 5, 19, 10, 0, 0, 0, time.UTC),
				DiskPath:  filepath.Join(runtimeDir, "snapshots", "clean.qcow2"),
			},
		},
	}
	if err := state.Save(paths.StateFile, current); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	_, err = service.Snapshot(context.Background(), SnapshotOptions{
		ProjectRoot: root,
		Target:      "web",
		Name:        "clean",
	})
	assertAppErrorCode(t, err, ErrorCodeConflict)
}

func TestSnapshotUsesRuntimeConflictAsConflict(t *testing.T) {
	service, root, _, fakeRuntime := newSnapshotServiceWithStoppedInstance(t)
	fakeRuntime.createSnapshotErr = errors.New("snapshot file already exists")

	_, err := service.Snapshot(context.Background(), SnapshotOptions{
		ProjectRoot: root,
		Target:      "web",
		Name:        "clean",
	})
	assertAppErrorCode(t, err, ErrorCodeConflict)
}

func TestSnapshotRequiresValidName(t *testing.T) {
	service, root, _, _ := newSnapshotServiceWithStoppedInstance(t)

	_, err := service.Snapshot(context.Background(), SnapshotOptions{
		ProjectRoot: root,
		Target:      "web",
		Name:        "../bad",
	})
	assertAppErrorCode(t, err, ErrorCodeInvalidArgument)
}

func newSnapshotServiceWithStoppedInstance(t *testing.T) (*Service, string, project.Metadata, *fakeSnapshotRuntime) {
	t.Helper()

	root := t.TempDir()
	yeastHome := filepath.Join(root, "yeast-home")
	service := NewService()
	service.resolveYeastHome = func() (string, error) { return yeastHome, nil }
	fakeRuntime := &fakeSnapshotRuntime{}
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
	runtimeDir, err := paths.InstanceDir("web")
	if err != nil {
		t.Fatalf("InstanceDir returned error: %v", err)
	}
	if err := os.MkdirAll(runtimeDir, 0755); err != nil {
		t.Fatalf("mkdir runtime dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(runtimeDir, "disk.qcow2"), []byte("disk-data"), 0644); err != nil {
		t.Fatalf("write instance disk: %v", err)
	}

	current := state.New(metadata.ID)
	current.Instances["web"] = state.InstanceState{
		Status:     "stopped",
		RuntimeDir: runtimeDir,
	}
	if err := state.Save(paths.StateFile, current); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	return service, root, metadata, fakeRuntime
}

type fakeSnapshotRuntime struct {
	snapshotPlan      rtm.SnapshotPlan
	createSnapshotErr error
}

func (f *fakeSnapshotRuntime) PrepareDisk(ctx context.Context, plan rtm.MachinePlan) (rtm.DiskPlan, error) {
	return plan.Disk, nil
}

func (f *fakeSnapshotRuntime) Start(ctx context.Context, plan rtm.MachinePlan) (rtm.RuntimeInstance, error) {
	return rtm.RuntimeInstance{}, nil
}

func (f *fakeSnapshotRuntime) Stop(ctx context.Context, instance rtm.RuntimeInstance, timeout time.Duration) error {
	return nil
}

func (f *fakeSnapshotRuntime) Inspect(ctx context.Context, instance rtm.RuntimeInstance) (rtm.ProcessInfo, error) {
	return rtm.ProcessInfo{}, nil
}

func (f *fakeSnapshotRuntime) CreateSnapshot(ctx context.Context, plan rtm.SnapshotPlan) error {
	f.snapshotPlan = plan
	return f.createSnapshotErr
}

func (f *fakeSnapshotRuntime) RestoreSnapshot(ctx context.Context, plan rtm.SnapshotPlan) error {
	return nil
}

func (f *fakeSnapshotRuntime) DeleteSnapshot(ctx context.Context, snapshotPath string) error {
	return nil
}

func (f *fakeSnapshotRuntime) Destroy(ctx context.Context, instance rtm.RuntimeInstance) error {
	return nil
}

func TestSnapshotMissingInstanceReturnsNotFound(t *testing.T) {
	service, root, metadata, _ := newSnapshotServiceWithStoppedInstance(t)

	paths, err := project.NewPaths(filepath.Join(root, "yeast-home"), metadata)
	if err != nil {
		t.Fatalf("NewPaths returned error: %v", err)
	}
	current := state.New(metadata.ID)
	if err := state.Save(paths.StateFile, current); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	_, err = service.Snapshot(context.Background(), SnapshotOptions{
		ProjectRoot: root,
		Target:      "api",
		Name:        "clean",
	})
	assertAppErrorCode(t, err, ErrorCodeNotFound)
	if !strings.Contains(err.Error(), `instance "api" not found`) {
		t.Fatalf("unexpected error: %v", err)
	}
}
