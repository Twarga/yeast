package app

import (
	"context"
	"errors"
	"github.com/Twarga/yeast/internal/guest"
	"github.com/Twarga/yeast/internal/project"
	provssh "github.com/Twarga/yeast/internal/provision/ssh"
	rtm "github.com/Twarga/yeast/internal/runtime"
	"github.com/Twarga/yeast/internal/state"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
	if !strings.Contains(gotRequest.Command, "whoami") || !strings.HasPrefix(gotRequest.Command, "timeout --kill-after=3s") {
		t.Fatalf("unexpected remote command: %q", gotRequest.Command)
	}
	if gotRequest.Timeout != defaultReadinessTimeout+30*time.Second {
		t.Fatalf("expected exec timeout %s, got %s", defaultReadinessTimeout+30*time.Second, gotRequest.Timeout)
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

	_, err := service.Exec(context.Background(), ExecOptions{
		GuestTargetOptions: GuestTargetOptions{ProjectRoot: t.TempDir()},
	})
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

// --- Task 4.1: agent-grade exec ------------------------------------------------

func newExecTestService(t *testing.T, run func(ctx context.Context, req provssh.RunRequest) (provssh.RunResult, error)) (*Service, string) {
	t.Helper()
	root := t.TempDir()
	yeastHome := filepath.Join(root, "yeast-home")
	service := NewService()
	service.resolveYeastHome = func() (string, error) { return yeastHome, nil }
	service.runtime = &fakeStatusRuntime{states: map[int]rtm.ProcessState{4242: rtm.ProcessStateRunning}}
	service.waitForTCP = func(ctx context.Context, options guest.ReadinessOptions) error { return nil }
	service.provisionTransport = provssh.FakeTransport{RunFunc: run}
	if _, err := service.Init(InitOptions{ProjectRoot: root}); err != nil {
		t.Fatal(err)
	}
	metadata, err := project.LoadMetadata(root)
	if err != nil {
		t.Fatal(err)
	}
	paths, err := project.NewPaths(yeastHome, metadata)
	if err != nil {
		t.Fatal(err)
	}
	current := state.New(metadata.ID)
	current.Instances["web"] = state.InstanceState{Status: "running", PID: 4242, SSHPort: 2205}
	if err := state.Save(paths.StateFile, current); err != nil {
		t.Fatal(err)
	}
	return service, root
}

func TestExecWrapsCwdEnvUserAndTimeout(t *testing.T) {
	var got provssh.RunRequest
	service, root := newExecTestService(t, func(ctx context.Context, req provssh.RunRequest) (provssh.RunResult, error) {
		got = req
		return provssh.RunResult{ExitCode: 0}, nil
	})
	_, err := service.Exec(context.Background(), ExecOptions{
		GuestTargetOptions: GuestTargetOptions{ProjectRoot: root, Target: "web"},
		Command:            []string{"make", "test"},
		Timeout:            30 * time.Second,
		Cwd:                "/srv/app",
		Env:                map[string]string{"A": "1", "B": "two words"},
		User:               "nobody",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"timeout --kill-after=3s 30s",
		"sudo -n -u 'nobody'",
		`cd '"'"'/srv/app'"'"'`,
		`env '"'"'A=1'"'"' '"'"'B=two words'"'"'`,
	} {
		if !strings.Contains(got.Command, want) {
			t.Fatalf("remote command missing %q:\n%s", want, got.Command)
		}
	}
	// order: cd before env before the command, all inside sh -c
	cdIdx := strings.Index(got.Command, "cd '")
	envIdx := strings.Index(got.Command, "env '")
	cmdIdx := strings.Index(got.Command, "'make'")
	if !(cdIdx >= 0 && envIdx > cdIdx && cmdIdx > envIdx) {
		t.Fatalf("wrong wrapper order:\n%s", got.Command)
	}
	// local timeout gets headroom beyond the guest timeout
	if got.Timeout <= 30*time.Second {
		t.Fatalf("local timeout %v not > guest timeout", got.Timeout)
	}
}

func TestExecGuestTimeoutExit124MarksTimedOut(t *testing.T) {
	service, root := newExecTestService(t, func(ctx context.Context, req provssh.RunRequest) (provssh.RunResult, error) {
		return provssh.RunResult{ExitCode: 124, Stderr: ""}, nil
	})
	result, err := service.Exec(context.Background(), ExecOptions{
		GuestTargetOptions: GuestTargetOptions{ProjectRoot: root, Target: "web"},
		Command:            []string{"sleep", "999"},
		Timeout:            time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Run.TimedOut {
		t.Fatal("expected TimedOut on guest-side exit 124")
	}
}

func TestExecCapsOutputAndWritesFullLog(t *testing.T) {
	big := strings.Repeat("x", 70*1024)
	service, root := newExecTestService(t, func(ctx context.Context, req provssh.RunRequest) (provssh.RunResult, error) {
		return provssh.RunResult{Stdout: big, Stderr: "warn", ExitCode: 0}, nil
	})
	result, err := service.Exec(context.Background(), ExecOptions{
		GuestTargetOptions: GuestTargetOptions{ProjectRoot: root, Target: "web"},
		Command:            []string{"big"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Run.Truncated {
		t.Fatal("expected Truncated")
	}
	if len(result.Run.Stdout) > 70*1024 {
		t.Fatalf("stdout not capped: %d", len(result.Run.Stdout))
	}
	if result.Run.FullOutputPath == "" {
		t.Fatal("missing FullOutputPath")
	}
	data, err := os.ReadFile(result.Run.FullOutputPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), big[:100]) || len(data) < 70*1024 {
		t.Fatal("full log missing untruncated output")
	}
	// stderr below cap stays untouched in-band... wait it keeps tail anyway
	if !strings.Contains(result.Run.Stderr, "warn") {
		t.Fatalf("stderr lost: %q", result.Run.Stderr)
	}
}

func TestExecPipesStdinThrough(t *testing.T) {
	var got provssh.RunRequest
	service, root := newExecTestService(t, func(ctx context.Context, req provssh.RunRequest) (provssh.RunResult, error) {
		got = req
		return provssh.RunResult{Stdout: "ok"}, nil
	})
	_, err := service.Exec(context.Background(), ExecOptions{
		GuestTargetOptions: GuestTargetOptions{ProjectRoot: root, Target: "web"},
		Command:            []string{"cat"},
		Stdin:              []byte("hello guest\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if string(got.Stdin) != "hello guest\n" {
		t.Fatalf("stdin = %q", got.Stdin)
	}
}
