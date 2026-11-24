package app

import (
	"context"
	"errors"
	"fmt"
	"github.com/Twarga/yeast/internal/config"
	"github.com/Twarga/yeast/internal/guest"
	"github.com/Twarga/yeast/internal/project"
	provssh "github.com/Twarga/yeast/internal/provision/ssh"
	rtm "github.com/Twarga/yeast/internal/runtime"
	"github.com/Twarga/yeast/internal/state"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func (s *Service) Exec(ctx context.Context, options ExecOptions) (out ExecResult, err error) {
	started := time.Now()
	root := options.ProjectRoot
	if root == "" {
		root = "."
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return ExecResult{}, WrapError(ErrorCodeInternal, fmt.Sprintf("resolve project root: %v", err), err)
	}
	defer func() {
		s.auditCall(ctx, absoluteRoot, "exec", map[string]any{"instance": options.Target, "command": strings.Join(options.Command, " "), "cwd": options.Cwd, "user": options.User}, started, err)
	}()
	if len(options.Command) == 0 {
		return ExecResult{}, WrapError(ErrorCodeInvalidArgument, "exec command is required", nil)
	}

	metadata, err := project.LoadMetadata(absoluteRoot)
	if err != nil {
		if errors.Is(err, project.ErrMetadataNotFound) {
			return ExecResult{}, WrapError(ErrorCodePrecondition, err.Error(), err)
		}
		return ExecResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	yeastHome, err := s.resolveYeastHome()
	if err != nil {
		return ExecResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	paths, err := project.NewPaths(yeastHome, metadata)
	if err != nil {
		return ExecResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}

	lock, err := state.Acquire(paths.StateLock, state.DefaultLockOptions())
	if err != nil {
		return ExecResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	defer func() { _ = lock.Release() }()

	cfg, err := config.Load(filepath.Join(absoluteRoot, ConfigFileName))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ExecResult{}, WrapError(ErrorCodePrecondition, err.Error(), err)
		}
		return ExecResult{}, WrapError(ErrorCodeInvalidArgument, err.Error(), err)
	}
	currentState, err := state.Load(paths.StateFile, metadata.ID)
	if err != nil {
		return ExecResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	changed := state.Reconcile(&currentState, state.ReconcileOptions{
		IsProcessAlive: func(pid int) bool {
			info, err := s.runtime.Inspect(ctx, rtm.RuntimeInstance{PID: pid})
			if err != nil {
				return false
			}
			return info.State == rtm.ProcessStateRunning
		},
	})
	if changed {
		if err := state.Save(paths.StateFile, currentState); err != nil {
			return ExecResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
		}
	}

	selectedName, selectedState, err := selectSSHInstance(currentState, options.Target)
	if err != nil {
		return ExecResult{}, err
	}
	sshUser := selectedState.User
	if instanceCfg, ok := lookupConfigInstance(cfg, selectedName); ok && instanceCfg.User != "" {
		sshUser = instanceCfg.User
	} else if selectedState.ForkOf != "" {
		if parentCfg, ok := lookupConfigInstance(cfg, selectedState.ForkOf); ok && parentCfg.User != "" && sshUser == "" {
			sshUser = parentCfg.User
		}
	}
	if sshUser == "" {
		sshUser = "yeast"
	}
	managementHost := resolveManagementHost(cfg)
	address, err := s.sshAddress(managementHost, selectedState.SSHPort)
	if err != nil {
		return ExecResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	if err := s.waitForTCP(ctx, guest.ReadinessOptions{
		Address: address,
		Timeout: defaultReadinessTimeout,
	}); err != nil {
		return ExecResult{}, WrapError(ErrorCodePrecondition, fmt.Sprintf("wait for ssh readiness for %s: %v", selectedName, err), err)
	}

	transport := s.provisionTransport
	if transport == nil {
		transport = provssh.NewLocalTransport()
	}

	timeout := options.Timeout
	if timeout <= 0 {
		timeout = defaultReadinessTimeout
	}

	displayCommand := commandString(options.Command)
	remoteCommand := buildRemoteExecCommand(options, timeout)
	// The guest-side `timeout` kills the whole process group on expiry; the
	// local transport deadline is only a backstop, so give it headroom.
	localTimeout := timeout + 30*time.Second
	runResult, runErr := transport.Run(ctx, provssh.RunRequest{
		User:    sshUser,
		Host:    managementHost,
		Port:    selectedState.SSHPort,
		Command: remoteCommand,
		Timeout: localTimeout,
		Stdin:   options.Stdin,
	})

	finishedAt := time.Now().UTC()
	startedAt := finishedAt.Add(-runResult.Duration)
	timedOut := errors.Is(runErr, context.DeadlineExceeded) ||
		((runResult.ExitCode == 124 || runResult.ExitCode == 137) && timeout > 0)
	if timedOut {
		runResult.ExitCode = 124
	}
	result := ExecResult{
		ProjectID: metadata.ID,
		Instance:  selectedName,
		Run: GuestCommandResult{
			Command:    displayCommand,
			ExitCode:   runResult.ExitCode,
			Stdout:     runResult.Stdout,
			Stderr:     runResult.Stderr,
			StartedAt:  startedAt,
			FinishedAt: finishedAt,
			Duration:   runResult.Duration,
			TimedOut:   timedOut,
		},
	}
	// Agent-grade output cap: keep the 64 KiB tail in-band, persist the full
	// payload under the project runtime dir so nothing is lost.
	if len(runResult.Stdout) > execOutputCapBytes || len(runResult.Stderr) > execOutputCapBytes {
		fullPath := writeFullExecOutput(paths, selectedName, displayCommand, runResult)
		result.Run.Stdout = tailString(runResult.Stdout, execOutputCapBytes)
		result.Run.Stderr = tailString(runResult.Stderr, execOutputCapBytes)
		result.Run.Truncated = true
		result.Run.FullOutputPath = fullPath
	}

	if runErr != nil {
		if result.Run.TimedOut {
			return result, nil
		}
		if runResult.ExitCode > 0 && runResult.ExitCode != 255 {
			return result, nil
		}
		return ExecResult{}, WrapError(ErrorCodeGuest, fmt.Sprintf("exec on instance %s: %v", selectedName, runErr), runErr)
	}

	return result, nil
}

const execOutputCapBytes = 64 * 1024

// buildRemoteExecCommand wraps the user command with cwd/env/sudo and a
// guest-side `timeout` so expiry kills the entire remote process group —
// not just the local ssh client.
func buildRemoteExecCommand(options ExecOptions, timeout time.Duration) string {
	inner := ""
	if options.Cwd != "" {
		inner = "cd " + shellQuote(options.Cwd) + " && "
	}
	if len(options.Env) > 0 {
		keys := make([]string, 0, len(options.Env))
		for k := range options.Env {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, k := range keys {
			parts = append(parts, shellQuote(fmt.Sprintf("%s=%s", k, options.Env[k])))
		}
		inner += "env " + strings.Join(parts, " ") + " "
	}
	inner += shellQuoteCommand(options.Command)
	wrapped := "sh -c " + shellQuote(inner)
	if options.User != "" {
		wrapped = "sudo -n -u " + shellQuote(options.User) + " " + wrapped
	}
	if timeout > 0 {
		secs := int(timeout.Seconds())
		if secs < 1 {
			secs = 1
		}
		wrapped = fmt.Sprintf("timeout --kill-after=3s %ds %s", secs, wrapped)
	}
	return wrapped
}

// writeFullExecOutput persists the untruncated output under the project
// runtime dir (logs/exec-<instance>-<ts>.log). Best-effort: on write
// failure the path is left empty but output is still capped.
func writeFullExecOutput(paths project.Paths, instance, command string, run provssh.RunResult) string {
	dir := filepath.Join(paths.ProjectDir, "logs")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return ""
	}
	name := fmt.Sprintf("exec-%s-%d.log", instance, time.Now().UTC().UnixNano())
	full := filepath.Join(dir, name)
	content := fmt.Sprintf("$ %s\n\n--- stdout ---\n%s\n--- stderr ---\n%s\n",
		command, run.Stdout, run.Stderr)
	if err := os.WriteFile(full, []byte(content), 0600); err != nil {
		return ""
	}
	return full
}

// tailString returns the last n bytes of s, cutting on a line boundary when
// one is nearby so the returned tail is readable.
func tailString(s string, n int) string {
	if len(s) <= n {
		return s
	}
	tail := s[len(s)-n:]
	if idx := strings.IndexByte(tail, '\n'); idx >= 0 && idx < len(tail)-1 {
		tail = tail[idx+1:]
	}
	return "[…truncated, full output in log file…]\n" + tail
}

func shellQuoteCommand(parts []string) string {
	quoted := make([]string, 0, len(parts))
	for _, part := range parts {
		quoted = append(quoted, shellQuote(part))
	}
	return strings.Join(quoted, " ")
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}
