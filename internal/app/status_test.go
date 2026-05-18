package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"yeast/internal/project"
	rtm "yeast/internal/runtime"
	"yeast/internal/state"
)

func TestStatusSortsInstancesByName(t *testing.T) {
	root := t.TempDir()
	yeastHome := filepath.Join(root, "yeast-home")
	service := NewService()
	service.resolveYeastHome = func() (string, error) { return yeastHome, nil }
	service.runtime = &fakeStatusRuntime{
		states: map[int]rtm.ProcessState{
			200: rtm.ProcessStateRunning,
			100: rtm.ProcessStateRunning,
		},
	}

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
	current.Instances["web"] = state.InstanceState{Status: "running", PID: 200}
	current.Instances["api"] = state.InstanceState{Status: "running", PID: 100}
	if err := state.Save(paths.StateFile, current); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	result, err := service.Status(context.Background(), StatusOptions{ProjectRoot: root})
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if len(result.Instances) != 2 {
		t.Fatalf("expected 2 instances, got %d", len(result.Instances))
	}
	if result.Instances[0].Name != "api" || result.Instances[1].Name != "web" {
		t.Fatalf("expected sorted instances, got %#v", result.Instances)
	}
}

func TestStatusReconcilesDeadProcessesAndSavesState(t *testing.T) {
	root := t.TempDir()
	yeastHome := filepath.Join(root, "yeast-home")
	service := NewService()
	service.resolveYeastHome = func() (string, error) { return yeastHome, nil }
	service.runtime = &fakeStatusRuntime{
		states: map[int]rtm.ProcessState{
			3333: rtm.ProcessStateStopped,
		},
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
	current.Instances["web"] = state.InstanceState{
		Status:             "running",
		PID:                3333,
		ManagementIP:       "127.0.0.1",
		SSHPort:            2222,
		ProvisionLogPath:   filepath.Join(yeastHome, "projects", metadata.ID, "instances", "web", "provision.log"),
		ProvisioningStatus: state.ProvisioningStatusNotStarted,
	}
	if err := state.Save(paths.StateFile, current); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	result, err := service.Status(context.Background(), StatusOptions{ProjectRoot: root})
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if result.Instances[0].Status != "stopped" {
		t.Fatalf("expected stopped status, got %q", result.Instances[0].Status)
	}
	if result.Instances[0].PID != 0 || result.Instances[0].SSHPort != 0 {
		t.Fatalf("expected pid and ssh port cleared, got %#v", result.Instances[0])
	}
	if result.Instances[0].ProvisionLogPath == "" {
		t.Fatalf("expected provision log path to remain set, got %#v", result.Instances[0])
	}
	if result.Instances[0].ProvisioningStatus != state.ProvisioningStatusNotStarted {
		t.Fatalf("expected provisioning status not_started, got %q", result.Instances[0].ProvisioningStatus)
	}

	reloaded, err := state.Load(paths.StateFile, metadata.ID)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if reloaded.Instances["web"].Status != "stopped" {
		t.Fatalf("expected saved state to be reconciled, got %#v", reloaded.Instances["web"])
	}
}

func TestStatusClassifiesUninitializedProject(t *testing.T) {
	root := t.TempDir()
	service := NewService()

	_, err := service.Status(context.Background(), StatusOptions{ProjectRoot: root})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "project metadata not found") {
		t.Fatalf("unexpected error: %v", err)
	}
	assertAppErrorCode(t, err, ErrorCodePrecondition)
}

func TestStatusClassifiesProjectRootResolutionFailure(t *testing.T) {
	root := t.TempDir()
	blocker := filepath.Join(root, "not-a-directory")
	if err := os.WriteFile(blocker, []byte("x"), 0644); err != nil {
		t.Fatalf("write blocker: %v", err)
	}

	service := NewService()
	_, err := service.Status(context.Background(), StatusOptions{ProjectRoot: filepath.Join(blocker, "child")})
	assertAppErrorCode(t, err, ErrorCodeInternal)
}

func TestStatusClassifiesStateProjectMismatch(t *testing.T) {
	root := t.TempDir()
	yeastHome := filepath.Join(root, "yeast-home")
	service := NewService()
	service.resolveYeastHome = func() (string, error) { return yeastHome, nil }
	service.runtime = &fakeStatusRuntime{}

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

	otherState := state.New("other-project")
	if err := state.Save(paths.StateFile, otherState); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	_, err = service.Status(context.Background(), StatusOptions{ProjectRoot: root})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "state project id mismatch") {
		t.Fatalf("unexpected error: %v", err)
	}
	assertAppErrorCode(t, err, ErrorCodeInternal)
}

type fakeStatusRuntime struct {
	states map[int]rtm.ProcessState
}

func (f *fakeStatusRuntime) PrepareDisk(ctx context.Context, plan rtm.MachinePlan) (rtm.DiskPlan, error) {
	return plan.Disk, nil
}

func (f *fakeStatusRuntime) Start(ctx context.Context, plan rtm.MachinePlan) (rtm.RuntimeInstance, error) {
	return rtm.RuntimeInstance{}, nil
}

func (f *fakeStatusRuntime) Stop(ctx context.Context, instance rtm.RuntimeInstance, timeout time.Duration) error {
	return nil
}

func (f *fakeStatusRuntime) Inspect(ctx context.Context, instance rtm.RuntimeInstance) (rtm.ProcessInfo, error) {
	stateValue := f.states[instance.PID]
	return rtm.ProcessInfo{PID: instance.PID, State: stateValue}, nil
}

func (f *fakeStatusRuntime) Destroy(ctx context.Context, instance rtm.RuntimeInstance) error {
	return nil
}
