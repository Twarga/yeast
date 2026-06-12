package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
	"yeast/internal/project"
	rtm "yeast/internal/runtime"
	"yeast/internal/state"
)

func TestSSHUsesStoredPortForSingleRunningInstance(t *testing.T) {
	root := t.TempDir()
	yeastHome := filepath.Join(root, "yeast-home")

	service := NewService()
	service.resolveYeastHome = func() (string, error) { return yeastHome, nil }
	service.runtime = &fakeStatusRuntime{states: map[int]rtm.ProcessState{4242: rtm.ProcessStateRunning}}

	var gotArgs []string
	service.runSSH = func(ctx context.Context, args []string) error {
		gotArgs = append([]string(nil), args...)
		return nil
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
	current.Instances["web"] = state.InstanceState{
		Status:  "running",
		PID:     4242,
		SSHPort: 2222,
	}
	if err := state.Save(paths.StateFile, current); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	result, err := service.SSH(context.Background(), SSHOptions{ProjectRoot: root})
	if err != nil {
		t.Fatalf("SSH returned error: %v", err)
	}
	if result.InstanceName != "web" {
		t.Fatalf("unexpected instance name: %q", result.InstanceName)
	}
	wantArgs := []string{
		"-p", "2222",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"yeast@127.0.0.1",
	}
	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Fatalf("unexpected ssh args:\n got: %#v\nwant: %#v", gotArgs, wantArgs)
	}
}

func TestSSHVerbosePassesFlag(t *testing.T) {
	root := t.TempDir()
	yeastHome := filepath.Join(root, "yeast-home")

	service := NewService()
	service.resolveYeastHome = func() (string, error) { return yeastHome, nil }
	service.runtime = &fakeStatusRuntime{states: map[int]rtm.ProcessState{4242: rtm.ProcessStateRunning}}

	var gotArgs []string
	service.runSSH = func(ctx context.Context, args []string) error {
		gotArgs = append([]string(nil), args...)
		return nil
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
	current.Instances["web"] = state.InstanceState{
		Status:  "running",
		PID:     4242,
		SSHPort: 2222,
	}
	if err := state.Save(paths.StateFile, current); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	_, err = service.SSH(context.Background(), SSHOptions{ProjectRoot: root, Verbose: true})
	if err != nil {
		t.Fatalf("SSH returned error: %v", err)
	}
	wantArgs := []string{
		"-v",
		"-p", "2222",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"yeast@127.0.0.1",
	}
	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Fatalf("unexpected ssh args:\n got: %#v\nwant: %#v", gotArgs, wantArgs)
	}
}

func TestSSHErrorsOnMultipleRunningInstancesWithoutTarget(t *testing.T) {
	root := t.TempDir()
	yeastHome := filepath.Join(root, "yeast-home")

	service := NewService()
	service.resolveYeastHome = func() (string, error) { return yeastHome, nil }
	service.runtime = &fakeStatusRuntime{
		states: map[int]rtm.ProcessState{
			1001: rtm.ProcessStateRunning,
			1002: rtm.ProcessStateRunning,
		},
	}
	service.runSSH = func(ctx context.Context, args []string) error { return nil }

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
	current.Instances["api"] = state.InstanceState{Status: "running", PID: 1001, SSHPort: 2222}
	current.Instances["web"] = state.InstanceState{Status: "running", PID: 1002, SSHPort: 2223}
	if err := state.Save(paths.StateFile, current); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	_, err = service.SSH(context.Background(), SSHOptions{ProjectRoot: root})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "multiple running instances") {
		t.Fatalf("unexpected error: %v", err)
	}
	assertAppErrorCode(t, err, ErrorCodeInvalidArgument)
}

func TestSelectSSHInstanceClassifiesSelectionErrors(t *testing.T) {
	t.Parallel()

	current := state.New("project-test")
	current.Instances["web"] = state.InstanceState{Status: "stopped", SSHPort: 2222}

	_, _, err := selectSSHInstance(current, "missing")
	assertAppErrorCode(t, err, ErrorCodeNotFound)

	_, _, err = selectSSHInstance(current, "web")
	assertAppErrorCode(t, err, ErrorCodePrecondition)

	_, _, err = selectSSHInstance(current, "")
	assertAppErrorCode(t, err, ErrorCodePrecondition)
}

func TestSSHClassifiesUninitializedProject(t *testing.T) {
	service := NewService()

	_, err := service.SSH(context.Background(), SSHOptions{ProjectRoot: t.TempDir()})
	assertAppErrorCode(t, err, ErrorCodePrecondition)
	if !strings.Contains(err.Error(), "project metadata not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSSHClassifiesMissingConfig(t *testing.T) {
	root := t.TempDir()
	yeastHome := filepath.Join(root, "yeast-home")
	service := NewService()
	service.resolveYeastHome = func() (string, error) { return yeastHome, nil }

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
	current.Instances["web"] = state.InstanceState{Status: "running", PID: os.Getpid(), SSHPort: 2222}
	if err := state.Save(paths.StateFile, current); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	if err := os.Remove(filepath.Join(root, ConfigFileName)); err != nil {
		t.Fatalf("remove config: %v", err)
	}

	_, err = service.SSH(context.Background(), SSHOptions{ProjectRoot: root})
	assertAppErrorCode(t, err, ErrorCodePrecondition)
}

func TestSSHClassifiesSSHAddressFailure(t *testing.T) {
	root := t.TempDir()
	yeastHome := filepath.Join(root, "yeast-home")
	service := NewService()
	service.resolveYeastHome = func() (string, error) { return yeastHome, nil }
	service.runtime = &fakeStatusRuntime{states: map[int]rtm.ProcessState{4242: rtm.ProcessStateRunning}}
	service.sshAddress = func(host string, port int) (string, error) {
		return "", errors.New("bad ssh address")
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
	current.Instances["web"] = state.InstanceState{Status: "running", PID: 4242, SSHPort: 2222}
	if err := state.Save(paths.StateFile, current); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	_, err = service.SSH(context.Background(), SSHOptions{ProjectRoot: root})
	assertAppErrorCode(t, err, ErrorCodeInternal)
}

func TestSSHClassifiesRunSSHFailure(t *testing.T) {
	root := t.TempDir()
	yeastHome := filepath.Join(root, "yeast-home")
	service := NewService()
	service.resolveYeastHome = func() (string, error) { return yeastHome, nil }
	service.runtime = &fakeStatusRuntime{states: map[int]rtm.ProcessState{4242: rtm.ProcessStateRunning}}
	service.runSSH = func(ctx context.Context, args []string) error {
		return errors.New("ssh command failed")
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
	current.Instances["web"] = state.InstanceState{Status: "running", PID: 4242, SSHPort: 2222}
	if err := state.Save(paths.StateFile, current); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	_, err = service.SSH(context.Background(), SSHOptions{ProjectRoot: root})
	assertAppErrorCode(t, err, ErrorCodeInternal)
}

func assertAppErrorCode(t *testing.T, err error, want ErrorCode) {
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
