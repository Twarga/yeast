package app

import (
	"context"
	"errors"
	"github.com/Twarga/yeast/internal/project"
	rtm "github.com/Twarga/yeast/internal/runtime"
	"github.com/Twarga/yeast/internal/state"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func (f *fakeSnapshotRuntime) DeleteSnapshot(ctx context.Context, plan rtm.DeleteSnapshotPlan) error {
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

// --- live snapshot tests (3.3/3.4) -------------------------------------------

type fakeLiveRuntime struct {
	fakeSnapshotRuntime
	savedPlan    *rtm.LiveSnapshotPlan
	restoredPlan *rtm.LiveRestorePlan
	saveInfo     rtm.LiveSnapshotInfo
	saveErr      error
	restored     rtm.RuntimeInstance
	restoreErr   error
}

func (f *fakeLiveRuntime) SaveLiveSnapshot(ctx context.Context, plan rtm.LiveSnapshotPlan) (rtm.LiveSnapshotInfo, error) {
	f.savedPlan = &plan
	return f.saveInfo, f.saveErr
}

func (f *fakeLiveRuntime) RestoreLiveSnapshot(ctx context.Context, plan rtm.LiveRestorePlan) (rtm.RuntimeInstance, error) {
	f.restoredPlan = &plan
	return f.restored, f.restoreErr
}

func TestSnapshotRunningInstanceTakesLiveSnapshot(t *testing.T) {
	service, root, metadata, _ := newSnapshotServiceWithStoppedInstance(t)
	fake := &fakeLiveRuntime{saveInfo: rtm.LiveSnapshotInfo{Method: "mapped-ram", PauseMs: 42},
		restored: rtm.RuntimeInstance{PID: 9999}}
	service.runtime = fake

	// make the instance running + give it a boot plan
	paths, _ := project.NewPaths(filepath.Join(root, "yeast-home"), metadata)
	st, _ := state.Load(paths.StateFile, metadata.ID)
	inst := st.Instances["web"]
	inst.Status = "running"
	inst.PID = 4242
	st.Instances["web"] = inst
	state.Save(paths.StateFile, st)
	writeBootPlan(inst.RuntimeDir, rtm.MachinePlan{Name: "web", MemoryMiB: 512, RuntimeDir: inst.RuntimeDir})

	result, err := service.Snapshot(context.Background(), SnapshotOptions{
		ProjectRoot: root, Target: "web", Name: "live1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Snapshot.Kind != "live" || result.Snapshot.Method != "mapped-ram" || result.Snapshot.PauseMs != 42 {
		t.Fatalf("snapshot = %+v", result.Snapshot)
	}
	if fake.savedPlan == nil || fake.savedPlan.Plan.MemoryMiB != 512 {
		t.Fatalf("saved plan = %+v", fake.savedPlan)
	}
	// state records live snapshot + ActiveDisk
	reloaded, _ := state.Load(paths.StateFile, metadata.ID)
	got := reloaded.Instances["web"]
	if got.Snapshots["live1"].Kind != "live" {
		t.Fatalf("state snapshot kind = %+v", got.Snapshots)
	}
	if got.ActiveDisk == "" {
		t.Fatal("ActiveDisk not set after live snapshot")
	}
}

func TestSnapshotRunningWithoutBootPlanFails(t *testing.T) {
	service, root, metadata, _ := newSnapshotServiceWithStoppedInstance(t)
	fake := &fakeLiveRuntime{}
	service.runtime = fake
	paths, _ := project.NewPaths(filepath.Join(root, "yeast-home"), metadata)
	st, _ := state.Load(paths.StateFile, metadata.ID)
	inst := st.Instances["web"]
	inst.Status = "running"
	inst.PID = 1
	st.Instances["web"] = inst
	state.Save(paths.StateFile, st)

	_, err := service.Snapshot(context.Background(), SnapshotOptions{ProjectRoot: root, Target: "web", Name: "x"})
	assertAppErrorCode(t, err, ErrorCodePrecondition)
}

func TestRestoreLiveSnapshotRunning(t *testing.T) {
	service, root, metadata, _ := newSnapshotServiceWithStoppedInstance(t)
	fake := &fakeLiveRuntime{
		restored: rtm.RuntimeInstance{PID: 8888},
	}
	service.runtime = fake

	paths, _ := project.NewPaths(filepath.Join(root, "yeast-home"), metadata)
	st, _ := state.Load(paths.StateFile, metadata.ID)
	inst := st.Instances["web"]
	inst.Status = "running"
	inst.PID = 4242
	inst.Snapshots = map[string]state.SnapshotState{
		"live1": {Name: "live1", Kind: "live", Dir: filepath.Join(inst.RuntimeDir, "snapshots", "live1"),
			DiskPath: filepath.Join(inst.RuntimeDir, "snapshots", "live1", "base.qcow2")},
	}
	st.Instances["web"] = inst
	state.Save(paths.StateFile, st)

	_, err := service.Restore(context.Background(), RestoreOptions{
		ProjectRoot: root, Target: "web", Name: "live1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if fake.restoredPlan == nil {
		t.Fatal("RestoreLiveSnapshot not called")
	}
	reloaded, _ := state.Load(paths.StateFile, metadata.ID)
	got := reloaded.Instances["web"]
	if got.PID != 8888 || got.Status != "running" {
		t.Fatalf("state after live restore = %+v", got)
	}
}
