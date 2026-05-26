package app

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
	"yeast/internal/guest"
	"yeast/internal/project"
	provssh "yeast/internal/provision/ssh"
	rtm "yeast/internal/runtime"
	"yeast/internal/state"
)

func TestExecRunsCommandAndReturnsStructuredResult(t *testing.T) {
	root := t.TempDir()
	yeastHome := filepath.Join(root, "yeast-home")

	service := NewService()
	service.resolveYeastHome = func() (string, error) { return yeastHome, nil }
	service.runtime = &fakeStatusRuntime{states: map[int]rtm.ProcessState{4242: rtm.ProcessStateRunning}}
	service.waitForTCP = func(ctx context.Context, options guest.ReadinessOptions) error { return nil }

	var gotRequest provssh.RunRequest
	service.provisionTransport = provssh.FakeTransport{
		RunFunc: func(ctx context.Context, request provssh.RunRequest) (provssh.RunResult, error) {
			gotRequest = request
			return provssh.RunResult{
				Stdout:   "yeast\n",
				Stderr:   "",
				ExitCode: 0,
				Duration: 200 * time.Millisecond,
			}, nil
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
	current.Instances["web"] = state.InstanceState{Status: "running", PID: 4242, SSHPort: 2205}
	if err := state.Save(paths.StateFile, current); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	result, err := service.Exec(context.Background(), ExecOptions{
		GuestTargetOptions: GuestTargetOptions{ProjectRoot: root, Target: "web"},
		Command:            []string{"whoami"},
	})
	if err != nil {
		t.Fatalf("Exec returned error: %v", err)
	}
	if gotRequest.Command != "'whoami'" {
		t.Fatalf("unexpected remote command: %q", gotRequest.Command)
	}
	if result.Instance != "web" || result.Run.Stdout != "yeast\n" || result.Run.ExitCode != 0 {
		t.Fatalf("unexpected exec result: %#v", result)
	}
	if result.Run.Command != "whoami" {
		t.Fatalf("unexpected display command: %q", result.Run.Command)
	}
	if result.Run.Duration != 200*time.Millisecond {
		t.Fatalf("unexpected duration: %s", result.Run.Duration)
	}
}

func TestExecReturnsResultForRemoteNonZeroExit(t *testing.T) {
	root := t.TempDir()
	yeastHome := filepath.Join(root, "yeast-home")

	service := NewService()
	service.resolveYeastHome = func() (string, error) { return yeastHome, nil }
	service.runtime = &fakeStatusRuntime{states: map[int]rtm.ProcessState{4242: rtm.ProcessStateRunning}}
	service.waitForTCP = func(ctx context.Context, options guest.ReadinessOptions) error { return nil }
	service.provisionTransport = provssh.FakeTransport{
		RunFunc: func(ctx context.Context, request provssh.RunRequest) (provssh.RunResult, error) {
			return provssh.RunResult{
				Stdout:   "",
				Stderr:   "missing\n",
				ExitCode: 2,
				Duration: 100 * time.Millisecond,
			}, errors.New("ssh failed")
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
	current.Instances["web"] = state.InstanceState{Status: "running", PID: 4242, SSHPort: 2205}
	if err := state.Save(paths.StateFile, current); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	result, err := service.Exec(context.Background(), ExecOptions{
		GuestTargetOptions: GuestTargetOptions{ProjectRoot: root, Target: "web"},
		Command:            []string{"cat", "/missing"},
	})
	if err != nil {
		t.Fatalf("Exec returned unexpected error: %v", err)
	}
	if result.Run.ExitCode != 2 || result.Run.Stderr != "missing\n" {
		t.Fatalf("unexpected non-zero exec result: %#v", result)
	}
}

func TestExecMarksTimeoutWithoutTransportErrorSurface(t *testing.T) {
	root := t.TempDir()
	yeastHome := filepath.Join(root, "yeast-home")

	service := NewService()
	service.resolveYeastHome = func() (string, error) { return yeastHome, nil }
	service.runtime = &fakeStatusRuntime{states: map[int]rtm.ProcessState{4242: rtm.ProcessStateRunning}}
	service.waitForTCP = func(ctx context.Context, options guest.ReadinessOptions) error { return nil }
	service.provisionTransport = provssh.FakeTransport{
		RunFunc: func(ctx context.Context, request provssh.RunRequest) (provssh.RunResult, error) {
			return provssh.RunResult{
				Stdout:   "",
				Stderr:   "",
				ExitCode: 0,
				Duration: time.Second,
			}, context.DeadlineExceeded
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
	current.Instances["web"] = state.InstanceState{Status: "running", PID: 4242, SSHPort: 2205}
	if err := state.Save(paths.StateFile, current); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	result, err := service.Exec(context.Background(), ExecOptions{
		GuestTargetOptions: GuestTargetOptions{ProjectRoot: root, Target: "web"},
		Command:            []string{"sleep", "10"},
		Timeout:            time.Second,
	})
	if err != nil {
		t.Fatalf("Exec returned unexpected error: %v", err)
	}
	if !result.Run.TimedOut {
		t.Fatalf("expected timed out result, got %#v", result)
	}
}

func TestExecClassifiesTransportFailureAsGuestError(t *testing.T) {
	root := t.TempDir()
	yeastHome := filepath.Join(root, "yeast-home")

	service := NewService()
	service.resolveYeastHome = func() (string, error) { return yeastHome, nil }
	service.runtime = &fakeStatusRuntime{states: map[int]rtm.ProcessState{4242: rtm.ProcessStateRunning}}
	service.waitForTCP = func(ctx context.Context, options guest.ReadinessOptions) error { return nil }
	service.provisionTransport = provssh.FakeTransport{
		RunFunc: func(ctx context.Context, request provssh.RunRequest) (provssh.RunResult, error) {
			return provssh.RunResult{ExitCode: 255, Duration: 10 * time.Millisecond}, errors.New("ssh failed")
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
	current.Instances["web"] = state.InstanceState{Status: "running", PID: 4242, SSHPort: 2205}
	if err := state.Save(paths.StateFile, current); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	_, err = service.Exec(context.Background(), ExecOptions{
		GuestTargetOptions: GuestTargetOptions{ProjectRoot: root, Target: "web"},
		Command:            []string{"whoami"},
	})
	assertAppErrorCode(t, err, ErrorCodeGuest)
}

func TestExecRequiresCommand(t *testing.T) {
	service := NewService()

	_, err := service.Exec(context.Background(), ExecOptions{})
	assertAppErrorCode(t, err, ErrorCodeInvalidArgument)
}

func TestShellQuoteCommandEscapesUnsafeParts(t *testing.T) {
	t.Parallel()

	got := shellQuoteCommand([]string{"bash", "-lc", "echo 'hi there'"})
	want := "'bash' '-lc' 'echo '\"'\"'hi there'\"'\"''"
	if got != want {
		t.Fatalf("unexpected shell quoted command:\n got: %s\nwant: %s", got, want)
	}
}
