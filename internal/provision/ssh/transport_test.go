package ssh

import (
	"context"
	"errors"
	"reflect"
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
