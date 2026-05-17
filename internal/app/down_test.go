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

func TestDownStopsRunningVMsAndMarksStopped(t *testing.T) {
	root := t.TempDir()
	yeastHome := filepath.Join(root, "yeast-home")

	service := NewService()
	service.resolveYeastHome = func() (string, error) { return yeastHome, nil }

	fakeRuntime := &fakeDownRuntime{}
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
	current.Instances["api"] = state.InstanceState{Status: "running", PID: 1001, SSHPort: 2222, RuntimeDir: "/tmp/api"}
	current.Instances["web"] = state.InstanceState{Status: "running", PID: 1002, SSHPort: 2223, RuntimeDir: "/tmp/web"}
	if err := state.Save(paths.StateFile, current); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	result, err := service.Down(context.Background(), DownOptions{ProjectRoot: root, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("Down returned error: %v", err)
	}

	if len(result.Instances) != 2 {
		t.Fatalf("expected 2 instance results, got %d", len(result.Instances))
	}
	wantStopped := []int{1001, 1002}
	if !reflect.DeepEqual(fakeRuntime.stoppedPIDs, wantStopped) {
		t.Fatalf("unexpected stopped pids:\n got: %#v\nwant: %#v", fakeRuntime.stoppedPIDs, wantStopped)
	}

	reloaded, err := state.Load(paths.StateFile, metadata.ID)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	for name, instance := range reloaded.Instances {
		if instance.Status != "stopped" || instance.PID != 0 || instance.SSHPort != 0 {
			t.Fatalf("expected %s to be stopped, got %#v", name, instance)
		}
	}
}

func TestDownHandlesAlreadyStoppedInstances(t *testing.T) {
	root := t.TempDir()
	yeastHome := filepath.Join(root, "yeast-home")

	service := NewService()
	service.resolveYeastHome = func() (string, error) { return yeastHome, nil }
	service.runtime = &fakeDownRuntime{}

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
	current.Instances["web"] = state.InstanceState{Status: "stopped"}
	if err := state.Save(paths.StateFile, current); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	result, err := service.Down(context.Background(), DownOptions{ProjectRoot: root})
	if err != nil {
		t.Fatalf("Down returned error: %v", err)
	}
	if result.Instances[0].Status != "already_stopped" {
		t.Fatalf("expected already_stopped result, got %#v", result.Instances[0])
	}
}

func TestDownClassifiesRuntimeStopFailure(t *testing.T) {
	root := t.TempDir()
	yeastHome := filepath.Join(root, "yeast-home")

	service := NewService()
	service.resolveYeastHome = func() (string, error) { return yeastHome, nil }
	service.runtime = &fakeDownRuntime{stopErr: errors.New("stop failed")}

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
	current.Instances["web"] = state.InstanceState{Status: "running", PID: 1001, SSHPort: 2222, RuntimeDir: "/tmp/web"}
	if err := state.Save(paths.StateFile, current); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	_, err = service.Down(context.Background(), DownOptions{ProjectRoot: root})
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

func TestDownClassifiesUninitializedProject(t *testing.T) {
	service := NewService()

	_, err := service.Down(context.Background(), DownOptions{ProjectRoot: t.TempDir()})
	assertDownAppErrorCode(t, err, ErrorCodePrecondition)
}

func TestDownClassifiesStateProjectMismatch(t *testing.T) {
	root := t.TempDir()
	yeastHome := filepath.Join(root, "yeast-home")

	service := NewService()
	service.resolveYeastHome = func() (string, error) { return yeastHome, nil }
	service.runtime = &fakeDownRuntime{}

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

	_, err = service.Down(context.Background(), DownOptions{ProjectRoot: root})
	assertDownAppErrorCode(t, err, ErrorCodeInternal)
}

func assertDownAppErrorCode(t *testing.T, err error, want ErrorCode) {
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

type fakeDownRuntime struct {
	stoppedPIDs []int
	stopErr     error
}

func (f *fakeDownRuntime) PrepareDisk(ctx context.Context, plan rtm.MachinePlan) (rtm.DiskPlan, error) {
	return plan.Disk, nil
}

func (f *fakeDownRuntime) Start(ctx context.Context, plan rtm.MachinePlan) (rtm.RuntimeInstance, error) {
	return rtm.RuntimeInstance{}, nil
}

func (f *fakeDownRuntime) Stop(ctx context.Context, instance rtm.RuntimeInstance, timeout time.Duration) error {
	f.stoppedPIDs = append(f.stoppedPIDs, instance.PID)
	return f.stopErr
}

func (f *fakeDownRuntime) Inspect(ctx context.Context, instance rtm.RuntimeInstance) (rtm.ProcessInfo, error) {
	return rtm.ProcessInfo{}, nil
}

func (f *fakeDownRuntime) Destroy(ctx context.Context, instance rtm.RuntimeInstance) error {
	return nil
}
