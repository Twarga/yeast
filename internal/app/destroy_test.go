package app

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"testing"
	"time"
	"yeast/internal/project"
	rtm "yeast/internal/runtime"
	"yeast/internal/state"
)

func TestDestroyStopsAndRemovesTrackedInstances(t *testing.T) {
	root := t.TempDir()
	yeastHome := filepath.Join(root, "yeast-home")

	service := NewService()
	service.resolveYeastHome = func() (string, error) { return yeastHome, nil }

	fakeRuntime := &fakeDestroyRuntime{}
	service.runtime = fakeRuntime

	if _, err := service.Init(InitOptions{ProjectRoot: root, Now: time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC)}); err != nil {
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
	current.Instances["api"] = state.InstanceState{Status: "running", PID: 1001, RuntimeDir: "/tmp/api"}
	current.Instances["web"] = state.InstanceState{Status: "stopped", RuntimeDir: "/tmp/web"}
	if err := state.Save(paths.StateFile, current); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	result, err := service.Destroy(context.Background(), DestroyOptions{ProjectRoot: root, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("Destroy returned error: %v", err)
	}

	if len(result.Instances) != 2 {
		t.Fatalf("expected 2 instance results, got %d", len(result.Instances))
	}
	wantDestroyed := []rtm.RuntimeInstance{
		{Name: "api", RuntimeDir: "/tmp/api", PID: 1001},
		{Name: "web", RuntimeDir: "/tmp/web", PID: 0},
	}
	if !reflect.DeepEqual(fakeRuntime.destroyed, wantDestroyed) {
		t.Fatalf("unexpected destroyed instances:\n got: %#v\nwant: %#v", fakeRuntime.destroyed, wantDestroyed)
	}

	reloaded, err := state.Load(paths.StateFile, metadata.ID)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(reloaded.Instances) != 0 {
		t.Fatalf("expected no remaining state entries, got %#v", reloaded.Instances)
	}
}

func TestDestroyClassifiesRuntimeDestroyFailure(t *testing.T) {
	root := t.TempDir()
	yeastHome := filepath.Join(root, "yeast-home")

	service := NewService()
	service.resolveYeastHome = func() (string, error) { return yeastHome, nil }
	service.runtime = &fakeDestroyRuntime{destroyErr: errors.New("destroy failed")}

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
	current.Instances["web"] = state.InstanceState{Status: "running", PID: 1001, RuntimeDir: "/tmp/web"}
	if err := state.Save(paths.StateFile, current); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	_, err = service.Destroy(context.Background(), DestroyOptions{ProjectRoot: root})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var appErr *AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.Code != ErrorCodeInternal {
		t.Fatalf("expected internal error code, got %q", appErr.Code)
	}
}

func TestDestroyClassifiesUninitializedProject(t *testing.T) {
	service := NewService()

	_, err := service.Destroy(context.Background(), DestroyOptions{ProjectRoot: t.TempDir()})
	assertDestroyAppErrorCode(t, err, ErrorCodePrecondition)
}

func TestDestroyClassifiesStateProjectMismatch(t *testing.T) {
	root := t.TempDir()
	yeastHome := filepath.Join(root, "yeast-home")

	service := NewService()
	service.resolveYeastHome = func() (string, error) { return yeastHome, nil }
	service.runtime = &fakeDestroyRuntime{}

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

	if err := state.Save(paths.StateFile, state.New("wrong-project")); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	_, err = service.Destroy(context.Background(), DestroyOptions{ProjectRoot: root})
	assertDestroyAppErrorCode(t, err, ErrorCodeInternal)
}

func assertDestroyAppErrorCode(t *testing.T, err error, want ErrorCode) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var appErr *AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.Code != want {
		t.Fatalf("expected error code %q, got %q", want, appErr.Code)
	}
}

type fakeDestroyRuntime struct {
	destroyed  []rtm.RuntimeInstance
	destroyErr error
}

func (f *fakeDestroyRuntime) PrepareDisk(ctx context.Context, plan rtm.MachinePlan) (rtm.DiskPlan, error) {
	return plan.Disk, nil
}

func (f *fakeDestroyRuntime) Start(ctx context.Context, plan rtm.MachinePlan) (rtm.RuntimeInstance, error) {
	return rtm.RuntimeInstance{}, nil
}

func (f *fakeDestroyRuntime) Stop(ctx context.Context, instance rtm.RuntimeInstance, timeout time.Duration) error {
	return nil
}

func (f *fakeDestroyRuntime) Inspect(ctx context.Context, instance rtm.RuntimeInstance) (rtm.ProcessInfo, error) {
	return rtm.ProcessInfo{}, nil
}

func (f *fakeDestroyRuntime) CreateSnapshot(ctx context.Context, plan rtm.SnapshotPlan) error {
	return nil
}

func (f *fakeDestroyRuntime) RestoreSnapshot(ctx context.Context, plan rtm.SnapshotPlan) error {
	return nil
}

func (f *fakeDestroyRuntime) DeleteSnapshot(ctx context.Context, snapshotPath string) error {
	return nil
}

func (f *fakeDestroyRuntime) Destroy(ctx context.Context, instance rtm.RuntimeInstance) error {
	f.destroyed = append(f.destroyed, instance)
	return f.destroyErr
}
