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

func TestRestoreReplacesStoppedInstanceDiskFromSnapshot(t *testing.T) {
	service, root, metadata, fakeRuntime := newRestoreServiceWithStoppedSnapshot(t)

	result, err := service.Restore(context.Background(), RestoreOptions{
		ProjectRoot: root,
		Target:      "web",
		Name:        "clean",
	})
	if err != nil {
		t.Fatalf("Restore returned error: %v", err)
	}
	if result.ProjectID != metadata.ID {
		t.Fatalf("unexpected project id %q", result.ProjectID)
	}
	if result.Instance != "web" {
		t.Fatalf("unexpected restore instance %q", result.Instance)
	}
	if result.Snapshot.Name != "clean" {
		t.Fatalf("unexpected restore snapshot %#v", result.Snapshot)
	}
	if fakeRuntime.restorePlan.InstanceDiskPath == "" || fakeRuntime.restorePlan.SnapshotPath == "" {
		t.Fatalf("expected restore runtime plan to be populated, got %#v", fakeRuntime.restorePlan)
	}
}

func TestRestoreEmitsLifecycleEvents(t *testing.T) {
	service, root, metadata, _ := newRestoreServiceWithStoppedSnapshot(t)

	events := make([]Event, 0)
	_, err := service.Restore(context.Background(), RestoreOptions{
		ProjectRoot: root,
		Target:      "web",
		Name:        "clean",
		Events: func(event Event) {
			events = append(events, event)
		},
	})
	if err != nil {
		t.Fatalf("Restore returned error: %v", err)
	}

	got := eventNames(events)
	want := []EventName{
		EventProjectLoaded,
		EventRestoreStarted,
		EventRestoreFinished,
		EventWorkflowCompleted,
	}
	if strings.Join(eventNamesToStrings(got), "\n") != strings.Join(eventNamesToStrings(want), "\n") {
		t.Fatalf("unexpected events:\n got: %#v\nwant: %#v", got, want)
	}
	for _, event := range events {
		if event.ProjectID != metadata.ID {
			t.Fatalf("unexpected event project id: %#v", event)
		}
	}
}

func TestRestoreRequiresStoppedInstance(t *testing.T) {
	service, root, metadata, _ := newRestoreServiceWithStoppedSnapshot(t)

	paths, err := project.NewPaths(filepath.Join(root, "yeast-home"), metadata)
	if err != nil {
		t.Fatalf("NewPaths returned error: %v", err)
	}
	current := state.New(metadata.ID)
	runtimeDir := filepath.Join(root, "yeast-home", "projects", metadata.ID, "instances", "web")
	current.Instances["web"] = state.InstanceState{
		Status:     "running",
		PID:        os.Getpid(),
		RuntimeDir: runtimeDir,
		Snapshots: map[string]state.SnapshotState{
			"clean": {
				Name:      "clean",
				CreatedAt: time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC),
				DiskPath:  filepath.Join(runtimeDir, "snapshots", "clean.qcow2"),
			},
		},
	}
	if err := state.Save(paths.StateFile, current); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	_, err = service.Restore(context.Background(), RestoreOptions{
		ProjectRoot: root,
		Target:      "web",
		Name:        "clean",
	})
	assertAppErrorCode(t, err, ErrorCodePrecondition)
}

func TestRestoreMissingSnapshotReturnsNotFound(t *testing.T) {
	service, root, _, _ := newRestoreServiceWithStoppedSnapshot(t)

	_, err := service.Restore(context.Background(), RestoreOptions{
		ProjectRoot: root,
		Target:      "web",
		Name:        "missing",
	})
	assertAppErrorCode(t, err, ErrorCodeNotFound)
}

func TestRestoreMapsRuntimeMissingSnapshotToNotFound(t *testing.T) {
	service, root, _, fakeRuntime := newRestoreServiceWithStoppedSnapshot(t)
	fakeRuntime.restoreSnapshotErr = errors.New("inspect snapshot file /tmp/missing.qcow2: stat /tmp/missing.qcow2: no such file or directory")

	_, err := service.Restore(context.Background(), RestoreOptions{
		ProjectRoot: root,
		Target:      "web",
		Name:        "clean",
	})
	assertAppErrorCode(t, err, ErrorCodeNotFound)
}

func TestRestoreRequiresValidName(t *testing.T) {
	service, root, _, _ := newRestoreServiceWithStoppedSnapshot(t)

	_, err := service.Restore(context.Background(), RestoreOptions{
		ProjectRoot: root,
		Target:      "web",
		Name:        "../bad",
	})
	assertAppErrorCode(t, err, ErrorCodeInvalidArgument)
}

func newRestoreServiceWithStoppedSnapshot(t *testing.T) (*Service, string, project.Metadata, *fakeRestoreRuntime) {
	t.Helper()

	root := t.TempDir()
	yeastHome := filepath.Join(root, "yeast-home")
	service := NewService()
	service.resolveYeastHome = func() (string, error) { return yeastHome, nil }
	fakeRuntime := &fakeRestoreRuntime{}
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
	if err := os.MkdirAll(filepath.Join(runtimeDir, "snapshots"), 0755); err != nil {
		t.Fatalf("mkdir snapshots dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(runtimeDir, "disk.qcow2"), []byte("broken-data"), 0644); err != nil {
		t.Fatalf("write disk: %v", err)
	}
	snapshotPath := filepath.Join(runtimeDir, "snapshots", "clean.qcow2")
	if err := os.WriteFile(snapshotPath, []byte("clean-data"), 0644); err != nil {
		t.Fatalf("write snapshot: %v", err)
	}

	current := state.New(metadata.ID)
	current.Instances["web"] = state.InstanceState{
		Status:     "stopped",
		RuntimeDir: runtimeDir,
		Snapshots: map[string]state.SnapshotState{
			"clean": {
				Name:      "clean",
				CreatedAt: time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC),
				DiskPath:  snapshotPath,
			},
		},
	}
	if err := state.Save(paths.StateFile, current); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	return service, root, metadata, fakeRuntime
}

type fakeRestoreRuntime struct {
	restorePlan        rtm.SnapshotPlan
	restoreSnapshotErr error
}

func (f *fakeRestoreRuntime) PrepareDisk(ctx context.Context, plan rtm.MachinePlan) (rtm.DiskPlan, error) {
	return plan.Disk, nil
}

func (f *fakeRestoreRuntime) Start(ctx context.Context, plan rtm.MachinePlan) (rtm.RuntimeInstance, error) {
	return rtm.RuntimeInstance{}, nil
}

func (f *fakeRestoreRuntime) Stop(ctx context.Context, instance rtm.RuntimeInstance, timeout time.Duration) error {
	return nil
}

func (f *fakeRestoreRuntime) Inspect(ctx context.Context, instance rtm.RuntimeInstance) (rtm.ProcessInfo, error) {
	return rtm.ProcessInfo{}, nil
}

func (f *fakeRestoreRuntime) CreateSnapshot(ctx context.Context, plan rtm.SnapshotPlan) error {
	return nil
}

func (f *fakeRestoreRuntime) RestoreSnapshot(ctx context.Context, plan rtm.SnapshotPlan) error {
	f.restorePlan = plan
	return f.restoreSnapshotErr
}

func (f *fakeRestoreRuntime) DeleteSnapshot(ctx context.Context, snapshotPath string) error {
	return nil
}

func (f *fakeRestoreRuntime) Destroy(ctx context.Context, instance rtm.RuntimeInstance) error {
	return nil
}

func TestRestoreMissingInstanceReturnsNotFound(t *testing.T) {
	service, root, metadata, _ := newRestoreServiceWithStoppedSnapshot(t)

	paths, err := project.NewPaths(filepath.Join(root, "yeast-home"), metadata)
	if err != nil {
		t.Fatalf("NewPaths returned error: %v", err)
	}
	current := state.New(metadata.ID)
	if err := state.Save(paths.StateFile, current); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	_, err = service.Restore(context.Background(), RestoreOptions{
		ProjectRoot: root,
		Target:      "api",
		Name:        "clean",
	})
	assertAppErrorCode(t, err, ErrorCodeNotFound)
	if !strings.Contains(err.Error(), `instance "api" not found`) {
		t.Fatalf("unexpected error: %v", err)
	}
}
