package ssh

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

type stubRunner struct {
	command string
	args    []string
	result  CommandResult
	err     error
}

func (s *stubRunner) Run(ctx context.Context, command string, args []string) (CommandResult, error) {
	s.command = command
	s.args = append([]string(nil), args...)
	return s.result, s.err
}

type sequenceRunner struct {
	results []CommandResult
	errors  []error
	calls   int
}

func (s *sequenceRunner) Run(ctx context.Context, command string, args []string) (CommandResult, error) {
	idx := s.calls
	s.calls++
	if idx >= len(s.results) {
		idx = len(s.results) - 1
	}
	var err error
	if idx < len(s.errors) {
		err = s.errors[idx]
	}
	return s.results[idx], err
}

func TestLocalTransportRunBuildsSSHInvocation(t *testing.T) {
	runner := &stubRunner{
		result: CommandResult{
			Stdout:   "ok\n",
			Stderr:   "",
			ExitCode: 0,
			Duration: 10 * time.Millisecond,
		},
	}
	transport := NewLocalTransportWithRunner(runner)

	result, err := transport.Run(context.Background(), RunRequest{
		User:    "yeast",
		Host:    "127.0.0.1",
		Port:    2205,
		Command: "echo ready",
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if runner.command != "ssh" {
		t.Fatalf("expected ssh command, got %q", runner.command)
	}

	wantArgs := []string{
		"-p", "2205",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=10",
		"-o", "ConnectionAttempts=1",
		"yeast@127.0.0.1",
		"echo ready",
	}
	if !reflect.DeepEqual(runner.args, wantArgs) {
		t.Fatalf("unexpected ssh args:\n got: %#v\nwant: %#v", runner.args, wantArgs)
	}
	if result.Stdout != "ok\n" || result.ExitCode != 0 {
		t.Fatalf("unexpected run result: %#v", result)
	}
}

func TestLocalTransportUploadBuildsSCPInvocation(t *testing.T) {
	runner := &stubRunner{}
	transport := NewLocalTransportWithRunner(runner)

	err := transport.Upload(context.Background(), UploadRequest{
		User:        "yeast",
		Host:        "127.0.0.1",
		Port:        2205,
		Source:      "./site",
		Destination: "/srv/site",
	})
	if err != nil {
		t.Fatalf("Upload returned error: %v", err)
	}

	if runner.command != "scp" {
		t.Fatalf("expected scp command, got %q", runner.command)
	}

	wantArgs := []string{
		"-P", "2205",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=10",
		"-o", "ConnectionAttempts=1",
		"./site",
		"yeast@127.0.0.1:/srv/site",
	}
	if !reflect.DeepEqual(runner.args, wantArgs) {
		t.Fatalf("unexpected scp args:\n got: %#v\nwant: %#v", runner.args, wantArgs)
	}
}

func TestLocalTransportDownloadBuildsSCPInvocation(t *testing.T) {
	runner := &stubRunner{}
	transport := NewLocalTransportWithRunner(runner)

	err := transport.Download(context.Background(), DownloadRequest{
		User:        "yeast",
		Host:        "127.0.0.1",
		Port:        2205,
		Source:      "/srv/site/index.html",
		Destination: "./index.html",
	})
	if err != nil {
		t.Fatalf("Download returned error: %v", err)
	}

	if runner.command != "scp" {
		t.Fatalf("expected scp command, got %q", runner.command)
	}

	wantArgs := []string{
		"-P", "2205",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=10",
		"-o", "ConnectionAttempts=1",
		"yeast@127.0.0.1:/srv/site/index.html",
		"./index.html",
	}
	if !reflect.DeepEqual(runner.args, wantArgs) {
		t.Fatalf("unexpected scp args:\n got: %#v\nwant: %#v", runner.args, wantArgs)
	}
}

func TestLocalTransportRunValidatesRequest(t *testing.T) {
	transport := NewLocalTransportWithRunner(&stubRunner{})

	_, err := transport.Run(context.Background(), RunRequest{
		User:    "yeast",
		Host:    "127.0.0.1",
		Port:    2205,
		Command: "",
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestLocalTransportUploadValidatesRequest(t *testing.T) {
	transport := NewLocalTransportWithRunner(&stubRunner{})

	err := transport.Upload(context.Background(), UploadRequest{
		User:        "yeast",
		Host:        "127.0.0.1",
		Port:        2205,
		Source:      "",
		Destination: "/srv/site",
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestLocalTransportDownloadValidatesRequest(t *testing.T) {
	transport := NewLocalTransportWithRunner(&stubRunner{})

	err := transport.Download(context.Background(), DownloadRequest{
		User:        "yeast",
		Host:        "127.0.0.1",
		Port:        2205,
		Source:      "",
		Destination: "./artifact.txt",
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestLocalTransportPreservesRunnerResultOnFailure(t *testing.T) {
	runner := &stubRunner{
		result: CommandResult{
			Stdout:   "partial\n",
			Stderr:   "boom\n",
			ExitCode: 7,
			Duration: time.Millisecond,
		},
		err: errors.New("exit status 7"),
	}
	transport := NewLocalTransportWithRunner(runner)

	result, err := transport.Run(context.Background(), RunRequest{
		User:    "yeast",
		Host:    "127.0.0.1",
		Port:    2205,
		Command: "false",
	})
	if err == nil {
		t.Fatal("expected run error")
	}
	if result.ExitCode != 7 || result.Stderr != "boom\n" {
		t.Fatalf("expected runner result to survive error, got %#v", result)
	}
}

func TestLocalTransportRetriesTransientSSHFailure(t *testing.T) {
	runner := &sequenceRunner{
		results: []CommandResult{
			{Stderr: "kex_exchange_identification: read: Connection reset by peer\n", ExitCode: 255},
			{Stdout: "ok\n", ExitCode: 0},
		},
		errors: []error{errors.New("exit status 255"), nil},
	}
	transport := NewLocalTransportWithRunner(runner)

	result, err := transport.Run(context.Background(), RunRequest{
		User:    "yeast",
		Host:    "127.0.0.1",
		Port:    2205,
		Command: "echo ok",
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.Stdout != "ok\n" {
		t.Fatalf("unexpected result after retry: %#v", result)
	}
	if runner.calls != 2 {
		t.Fatalf("expected 2 attempts, got %d", runner.calls)
	}
}

func TestLocalTransportDoesNotRetryCommandFailure(t *testing.T) {
	runner := &sequenceRunner{
		results: []CommandResult{{Stderr: "command failed\n", ExitCode: 1}},
		errors:  []error{errors.New("exit status 1")},
	}
	transport := NewLocalTransportWithRunner(runner)

	_, err := transport.Run(context.Background(), RunRequest{
		User:    "yeast",
		Host:    "127.0.0.1",
		Port:    2205,
		Command: "false",
	})
	if err == nil {
		t.Fatal("expected run error")
	}
	if runner.calls != 1 {
		t.Fatalf("expected 1 attempt, got %d", runner.calls)
	}
}

func TestTransientSSHFailureRecognizesBannerTimeout(t *testing.T) {
	result := CommandResult{
		ExitCode: 255,
		Stderr:   "Connection timed out during banner exchange\nConnection to 127.0.0.1 port 2222 timed out\n",
	}
	if !isTransientSSHFailure(result) {
		t.Fatal("expected banner timeout to be transient")
	}
}

func TestFakeTransportHooks(t *testing.T) {
	transport := FakeTransport{
		RunFunc: func(ctx context.Context, request RunRequest) (RunResult, error) {
			return RunResult{Stdout: request.Command}, nil
		},
		UploadFunc: func(ctx context.Context, request UploadRequest) error {
			if request.Destination != "/srv/site" {
				t.Fatalf("unexpected destination: %q", request.Destination)
			}
			return nil
		},
	}

	result, err := transport.Run(context.Background(), RunRequest{Command: "echo ready"})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.Stdout != "echo ready" {
		t.Fatalf("unexpected fake run result: %#v", result)
	}
	if err := transport.Upload(context.Background(), UploadRequest{Destination: "/srv/site"}); err != nil {
		t.Fatalf("Upload returned error: %v", err)
	}
	if err := transport.Download(context.Background(), DownloadRequest{Destination: "./site"}); err != nil {
		t.Fatalf("Download returned error: %v", err)
	}
}

func TestMultiplexTransportIncludesControlMasterArgs(t *testing.T) {
	runner := &stubRunner{
		result: CommandResult{Stdout: "ok\n", ExitCode: 0, Duration: time.Millisecond},
	}
	transport := NewLocalTransportWithMultiplex(t.TempDir())
	transport.runner = runner
	defer transport.Close()

	_, err := transport.Run(context.Background(), RunRequest{
		User:    "yeast",
		Host:    "127.0.0.1",
		Port:    2205,
		Command: "echo hello",
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	hasControlMaster := false
	for i, arg := range runner.args {
		if arg == "ControlMaster=auto" && i > 0 && runner.args[i-1] == "-o" {
			hasControlMaster = true
			break
		}
	}
	if !hasControlMaster {
		t.Fatalf("expected ControlMaster=auto in args, got: %v", runner.args)
	}

	hasControlPath := false
	controlPath := ""
	controlPathIndex := -1
	destinationIndex := -1
	for i, arg := range runner.args {
		if i > 0 && runner.args[i-1] == "-o" && len(arg) > 12 && arg[:12] == "ControlPath=" {
			hasControlPath = true
			controlPath = strings.TrimPrefix(arg, "ControlPath=")
			controlPathIndex = i
		}
		if arg == "yeast@127.0.0.1" {
			destinationIndex = i
			break
		}
	}
	if !hasControlPath {
		t.Fatalf("expected ControlPath= in args, got: %v", runner.args)
	}
	if !strings.HasPrefix(controlPath, os.TempDir()) {
		t.Fatalf("expected short temp ControlPath, got %q", controlPath)
	}
	if len(controlPath)+24 > 108 {
		t.Fatalf("ControlPath leaves too little room for OpenSSH suffix: %q", controlPath)
	}
	if controlPathIndex < 0 || destinationIndex < 0 || controlPathIndex > destinationIndex {
		t.Fatalf("expected ControlPath before ssh destination, got: %v", runner.args)
	}
}
