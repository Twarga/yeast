package ssh

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"time"
)

type Transport interface {
	Run(ctx context.Context, request RunRequest) (RunResult, error)
	Upload(ctx context.Context, request UploadRequest) error
}

type Runner interface {
	Run(ctx context.Context, command string, args []string) (CommandResult, error)
}

type RunRequest struct {
	User    string
	Host    string
	Port    int
	Command string
	Timeout time.Duration
}

type RunResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
}

type UploadRequest struct {
	User        string
	Host        string
	Port        int
	Source      string
	Destination string
	Timeout     time.Duration
}

type CommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
}

type LocalTransport struct {
	runner Runner
}

func NewLocalTransport() *LocalTransport {
	return &LocalTransport{runner: OSRunner{}}
}

func NewLocalTransportWithRunner(runner Runner) *LocalTransport {
	return &LocalTransport{runner: runner}
}

func (t *LocalTransport) Run(ctx context.Context, request RunRequest) (RunResult, error) {
	if err := validateRunRequest(request); err != nil {
		return RunResult{}, err
	}
	runner := t.runner
	if runner == nil {
		runner = OSRunner{}
	}

	args := buildSSHBaseArgs(request.User, request.Host, request.Port)
	args = append(args, request.Command)
	runCtx := ctx
	cancel := func() {}
	if request.Timeout > 0 {
		runCtx, cancel = context.WithTimeout(ctx, request.Timeout)
	}
	defer cancel()

	result, err := runner.Run(runCtx, "ssh", args)
	if err != nil {
		return RunResult(result), err
	}

	return RunResult(result), nil
}

func (t *LocalTransport) Upload(ctx context.Context, request UploadRequest) error {
	if err := validateUploadRequest(request); err != nil {
		return err
	}
	runner := t.runner
	if runner == nil {
		runner = OSRunner{}
	}

	runCtx := ctx
	cancel := func() {}
	if request.Timeout > 0 {
		runCtx, cancel = context.WithTimeout(ctx, request.Timeout)
	}
	defer cancel()

	args := []string{
		"-P", strconv.Itoa(request.Port),
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		request.Source,
		fmt.Sprintf("%s@%s:%s", request.User, request.Host, request.Destination),
	}
	_, err := runner.Run(runCtx, "scp", args)
	return err
}

type OSRunner struct{}

func (OSRunner) Run(ctx context.Context, command string, args []string) (CommandResult, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	started := time.Now()
	err := cmd.Run()
	duration := time.Since(started)

	exitCode := 0
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}

	result := CommandResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
		Duration: duration,
	}
	if err != nil {
		return result, fmt.Errorf("%s failed: %w", command, err)
	}
	return result, nil
}

func buildSSHBaseArgs(user, host string, port int) []string {
	return []string{
		"-p", strconv.Itoa(port),
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		fmt.Sprintf("%s@%s", user, host),
	}
}

func validateRunRequest(request RunRequest) error {
	if request.User == "" {
		return fmt.Errorf("user is required")
	}
	if request.Host == "" {
		return fmt.Errorf("host is required")
	}
	if request.Port <= 0 {
		return fmt.Errorf("port must be greater than zero")
	}
	if request.Command == "" {
		return fmt.Errorf("command is required")
	}
	return nil
}

func validateUploadRequest(request UploadRequest) error {
	if request.User == "" {
		return fmt.Errorf("user is required")
	}
	if request.Host == "" {
		return fmt.Errorf("host is required")
	}
	if request.Port <= 0 {
		return fmt.Errorf("port must be greater than zero")
	}
	if request.Source == "" {
		return fmt.Errorf("source is required")
	}
	if request.Destination == "" {
		return fmt.Errorf("destination is required")
	}
	return nil
}
