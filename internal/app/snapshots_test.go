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

func TestListInstanceSnapshotsReturnsSortedSnapshots(t *testing.T) {
	currentState := state.New("proj_0123456789abcdef01234567")
	currentState.Instances["web"] = state.InstanceState{
		Snapshots: map[string]state.SnapshotState{
			"late": {
				Name:      "late",
				CreatedAt: time.Date(2026, 5, 18, 16, 0, 0, 0, time.UTC),
			},
			"early": {
				Name:      "early",
				CreatedAt: time.Date(2026, 5, 18, 14, 0, 0, 0, time.UTC),
			},
		},
	}

	got, err := listInstanceSnapshots(currentState, "web")
	if err != nil {
		t.Fatalf("listInstanceSnapshots returned error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 snapshots, got %d", len(got))
	}
	if got[0].Name != "early" || got[1].Name != "late" {
		t.Fatalf("unexpected snapshot order: %#v", got)
	}
}

func TestListInstanceSnapshotsHandlesMissingSnapshotState(t *testing.T) {
	currentState := state.New("proj_0123456789abcdef01234567")
	currentState.Instances["web"] = state.InstanceState{}

	got, err := listInstanceSnapshots(currentState, "web")
	if err != nil {
		t.Fatalf("listInstanceSnapshots returned error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty snapshot list, got %#v", got)
	}
}

func TestListInstanceSnapshotsRequiresTarget(t *testing.T) {
	_, err := listInstanceSnapshots(state.New("proj_0123456789abcdef01234567"), "")
	assertAppErrorCode(t, err, ErrorCodeInvalidArgument)
}

func TestListInstanceSnapshotsRequiresExistingInstance(t *testing.T) {
	_, err := listInstanceSnapshots(state.New("proj_0123456789abcdef01234567"), "web")
	assertAppErrorCode(t, err, ErrorCodeNotFound)
}

func TestSnapshotsReturnsTargetMetadata(t *testing.T) {
	service, root, metadata, _ := newSnapshotsServiceWithState(t)

	result, err := service.Snapshots(context.Background(), SnapshotsOptions{
		ProjectRoot: root,
		Target:      "web",
	})
	if err != nil {
		t.Fatalf("Snapshots returned error: %v", err)
	}
	if result.ProjectID != metadata.ID {
		t.Fatalf("unexpected project id %q", result.ProjectID)
	}
	if result.Instance != "web" {
		t.Fatalf("unexpected instance %q", result.Instance)
	}
	if len(result.Snapshots) != 2 {
		t.Fatalf("expected 2 snapshots, got %#v", result.Snapshots)
	}
	if result.Snapshots[0].Name != "clean" || result.Snapshots[1].Name != "late" {
		t.Fatalf("unexpected snapshot order: %#v", result.Snapshots)
	}
}

func TestSnapshotsMissingInstanceReturnsNotFound(t *testing.T) {
	service, root, _, _ := newSnapshotsServiceWithState(t)

	_, err := service.Snapshots(context.Background(), SnapshotsOptions{
		ProjectRoot: root,
		Target:      "api",
	})
	assertAppErrorCode(t, err, ErrorCodeNotFound)
}

func TestDeleteSnapshotRemovesFileAndMetadata(t *testing.T) {
	service, root, metadata, fakeRuntime := newSnapshotsServiceWithState(t)

	result, err := service.DeleteSnapshot(context.Background(), DeleteSnapshotOptions{
		ProjectRoot: root,
		Target:      "web",
		Name:        "clean",
	})
	if err != nil {
		t.Fatalf("DeleteSnapshot returned error: %v", err)
	}
	if result.ProjectID != metadata.ID {
		t.Fatalf("unexpected project id %q", result.ProjectID)
	}
	if result.Instance != "web" || result.Snapshot != "clean" {
		t.Fatalf("unexpected result %#v", result)
	}
	if !strings.HasSuffix(fakeRuntime.deleteSnapshotPath, "/clean.qcow2") {
		t.Fatalf("expected delete path to target clean snapshot, got %q", fakeRuntime.deleteSnapshotPath)
	}

	statePath := filepath.Join(root, "yeast-home", "projects", metadata.ID, "state.json")
	loaded, err := state.Load(statePath, metadata.ID)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if _, exists := loaded.Instances["web"].Snapshots["clean"]; exists {
		t.Fatalf("expected clean snapshot metadata to be removed, got %#v", loaded.Instances["web"].Snapshots)
	}
	if _, exists := loaded.Instances["web"].Snapshots["late"]; !exists {
		t.Fatalf("expected remaining snapshot metadata to be preserved, got %#v", loaded.Instances["web"].Snapshots)
	}
}

func TestDeleteSnapshotMissingSnapshotReturnsNotFound(t *testing.T) {
	service, root, _, _ := newSnapshotsServiceWithState(t)

	_, err := service.DeleteSnapshot(context.Background(), DeleteSnapshotOptions{
		ProjectRoot: root,
		Target:      "web",
		Name:        "missing",
	})
	assertAppErrorCode(t, err, ErrorCodeNotFound)
}

func TestDeleteSnapshotMapsRuntimeMissingFileToNotFound(t *testing.T) {
	service, root, _, fakeRuntime := newSnapshotsServiceWithState(t)
	fakeRuntime.deleteSnapshotErr = errors.New("remove /tmp/clean.qcow2: no such file or directory")

	_, err := service.DeleteSnapshot(context.Background(), DeleteSnapshotOptions{
		ProjectRoot: root,
		Target:      "web",
		Name:        "clean",
	})
	assertAppErrorCode(t, err, ErrorCodeNotFound)
}

func newSnapshotsServiceWithState(t *testing.T) (*Service, string, project.Metadata, *fakeSnapshotsRuntime) {
	t.Helper()

	root := t.TempDir()
	yeastHome := filepath.Join(root, "yeast-home")
	service := NewService()
	service.resolveYeastHome = func() (string, error) { return yeastHome, nil }
	fakeRuntime := &fakeSnapshotsRuntime{}
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
	snapshotsDir := filepath.Join(runtimeDir, "snapshots")
	if err := os.MkdirAll(snapshotsDir, 0755); err != nil {
		t.Fatalf("mkdir snapshots dir: %v", err)
	}

	current := state.New(metadata.ID)
	current.Instances["web"] = state.InstanceState{
		Status:     "stopped",
		RuntimeDir: runtimeDir,
		Snapshots: map[string]state.SnapshotState{
			"late": {
				Name:        "late",
				CreatedAt:   time.Date(2026, 5, 19, 13, 0, 0, 0, time.UTC),
				Description: "Later snapshot",
				DiskPath:    filepath.Join(snapshotsDir, "late.qcow2"),
			},
			"clean": {
				Name:        "clean",
				CreatedAt:   time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC),
				Description: "Clean baseline",
				DiskPath:    filepath.Join(snapshotsDir, "clean.qcow2"),
			},
		},
	}
	if err := state.Save(paths.StateFile, current); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	return service, root, metadata, fakeRuntime
}

type fakeSnapshotsRuntime struct {
	deleteSnapshotPath string
	deleteSnapshotErr  error
}

func (f *fakeSnapshotsRuntime) PrepareDisk(ctx context.Context, plan rtm.MachinePlan) (rtm.DiskPlan, error) {
	return plan.Disk, nil
}

func (f *fakeSnapshotsRuntime) Start(ctx context.Context, plan rtm.MachinePlan) (rtm.RuntimeInstance, error) {
	return rtm.RuntimeInstance{}, nil
}

func (f *fakeSnapshotsRuntime) Stop(ctx context.Context, instance rtm.RuntimeInstance, timeout time.Duration) error {
	return nil
}

func (f *fakeSnapshotsRuntime) Inspect(ctx context.Context, instance rtm.RuntimeInstance) (rtm.ProcessInfo, error) {
	return rtm.ProcessInfo{}, nil
}

func (f *fakeSnapshotsRuntime) CreateSnapshot(ctx context.Context, plan rtm.SnapshotPlan) error {
	return nil
}

func (f *fakeSnapshotsRuntime) RestoreSnapshot(ctx context.Context, plan rtm.SnapshotPlan) error {
	return nil
}

func (f *fakeSnapshotsRuntime) DeleteSnapshot(ctx context.Context, snapshotPath string) error {
	f.deleteSnapshotPath = snapshotPath
	return f.deleteSnapshotErr
}

func (f *fakeSnapshotsRuntime) Destroy(ctx context.Context, instance rtm.RuntimeInstance) error {
	return nil
}
