package ssh

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Transport interface {
	Run(ctx context.Context, request RunRequest) (RunResult, error)
	Upload(ctx context.Context, request UploadRequest) error
	Download(ctx context.Context, request DownloadRequest) error
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

type DownloadRequest struct {
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
	runner              Runner
	controlSocket       string
	controlSocketPrefix string
}

const transientSSHMaxAttempts = 24

func NewLocalTransport() *LocalTransport {
	return &LocalTransport{runner: OSRunner{}}
}

func NewLocalTransportWithRunner(runner Runner) *LocalTransport {
	return &LocalTransport{runner: runner}
}

func NewLocalTransportWithMultiplex(socketDir string) *LocalTransport {
	sum := sha256.Sum256([]byte(socketDir))
	prefix := fmt.Sprintf("yeast-ssh-%x", sum[:6])
	socketPath := filepath.Join(os.TempDir(), prefix+"-%p")
	return &LocalTransport{runner: OSRunner{}, controlSocket: socketPath, controlSocketPrefix: prefix}
}

func (t *LocalTransport) Close() {
	if t.controlSocket == "" {
		return
	}
	dir := filepath.Dir(t.controlSocket)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	prefix := t.controlSocketPrefix
	if prefix == "" {
		prefix = "ssh-mux"
	}
	for _, e := range entries {
		if name := e.Name(); strings.HasPrefix(name, prefix) {
			_ = os.Remove(filepath.Join(dir, name))
		}
	}
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
	args = append(args, t.controlArgs()...)
	args = append(args, fmt.Sprintf("%s@%s", request.User, request.Host))
	args = append(args, request.Command)
	result, err := runWithTransientSSHRetry(ctx, runner, "ssh", args, request.Timeout)
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

	args := []string{
		"-P", strconv.Itoa(request.Port),
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "LogLevel=ERROR",
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=10",
		"-o", "ConnectionAttempts=1",
	}
	args = append(args, t.scpControlArgs()...)
	args = append(args, request.Source, fmt.Sprintf("%s@%s:%s", request.User, request.Host, request.Destination))
	_, err := runWithTransientSSHRetry(ctx, runner, "scp", args, request.Timeout)
	return err
}

func (t *LocalTransport) Download(ctx context.Context, request DownloadRequest) error {
	if err := validateDownloadRequest(request); err != nil {
		return err
	}
	runner := t.runner
	if runner == nil {
		runner = OSRunner{}
	}

	args := []string{
		"-P", strconv.Itoa(request.Port),
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "LogLevel=ERROR",
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=10",
		"-o", "ConnectionAttempts=1",
	}
	args = append(args, t.scpControlArgs()...)
	args = append(args, fmt.Sprintf("%s@%s:%s", request.User, request.Host, request.Source), request.Destination)
	_, err := runWithTransientSSHRetry(ctx, runner, "scp", args, request.Timeout)
	return err
}

func runWithTransientSSHRetry(ctx context.Context, runner Runner, command string, args []string, timeout time.Duration) (CommandResult, error) {
	started := time.Now()
	var last CommandResult
	var lastErr error

	for attempt := 1; attempt <= transientSSHMaxAttempts; attempt++ {
		runCtx := ctx
		cancel := func() {}
		if timeout > 0 {
			remaining := timeout - time.Since(started)
			if remaining <= 0 {
				if lastErr != nil {
					return last, lastErr
				}
				return last, context.DeadlineExceeded
			}
			runCtx, cancel = context.WithTimeout(ctx, remaining)
		}

		result, err := runner.Run(runCtx, command, args)
		cancel()
		if err == nil {
			return result, nil
		}
		last, lastErr = result, err
		if !isTransientSSHFailure(result) || ctx.Err() != nil || attempt == transientSSHMaxAttempts {
			return result, err
		}
		delay := time.Duration(attempt) * 500 * time.Millisecond
		if delay > 3*time.Second {
			delay = 3 * time.Second
		}
		time.Sleep(delay)
	}
	return last, lastErr
}

func isTransientSSHFailure(result CommandResult) bool {
	if result.ExitCode != 255 {
		return false
	}
	text := strings.ToLower(result.Stderr + "\n" + result.Stdout)
	return strings.Contains(text, "connection reset") ||
		strings.Contains(text, "connection refused") ||
		strings.Contains(text, "kex_exchange_identification") ||
		strings.Contains(text, "connection closed") ||
		strings.Contains(text, "timed out during banner exchange") ||
		strings.Contains(text, "operation timed out")
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

func buildSSHBaseArgs(_, _ string, port int) []string {
	return []string{
		"-p", strconv.Itoa(port),
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "LogLevel=ERROR",
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=10",
		"-o", "ConnectionAttempts=1",
	}
}

func (t *LocalTransport) controlArgs() []string {
	if t.controlSocket == "" {
		return nil
	}
	return []string{
		"-o", "ControlMaster=auto",
		"-o", fmt.Sprintf("ControlPath=%s", t.controlSocket),
		"-o", "ControlPersist=60",
	}
}

func (t *LocalTransport) scpControlArgs() []string {
	if t.controlSocket == "" {
		return nil
	}
	return []string{
		"-o", "ControlMaster=auto",
		"-o", fmt.Sprintf("ControlPath=%s", t.controlSocket),
		"-o", "ControlPersist=60",
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

func validateDownloadRequest(request DownloadRequest) error {
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
