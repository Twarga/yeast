package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"yeast/internal/config"
	"yeast/internal/guest"
	"yeast/internal/project"
	provssh "yeast/internal/provision/ssh"
	rtm "yeast/internal/runtime"
	"yeast/internal/state"
)

func (s *Service) Exec(ctx context.Context, options ExecOptions) (ExecResult, error) {
	root := options.ProjectRoot
	if root == "" {
		root = "."
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return ExecResult{}, WrapError(ErrorCodeInternal, fmt.Sprintf("resolve project root: %v", err), err)
	}
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
	instanceCfg, ok := lookupConfigInstance(cfg, selectedName)
	if !ok {
		return ExecResult{}, WrapError(ErrorCodeNotFound, fmt.Sprintf("instance %q not found in config", selectedName), nil)
	}
	address, err := s.sshAddress(defaultManagementHost, selectedState.SSHPort)
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

	displayCommand := commandString(options.Command)
	runResult, runErr := transport.Run(ctx, provssh.RunRequest{
		User:    instanceCfg.User,
		Host:    defaultManagementHost,
		Port:    selectedState.SSHPort,
		Command: shellQuoteCommand(options.Command),
		Timeout: options.Timeout,
	})

	finishedAt := time.Now().UTC()
	startedAt := finishedAt.Add(-runResult.Duration)
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
			TimedOut:   errors.Is(runErr, context.DeadlineExceeded),
		},
	}

	if runErr != nil {
		if result.Run.TimedOut {
			return result, nil
		}
		if runResult.ExitCode > 0 && runResult.ExitCode != 255 {
			return result, nil
		}
		return ExecResult{}, WrapError(ErrorCodePrecondition, fmt.Sprintf("exec on instance %s: %v", selectedName, runErr), runErr)
	}

	return result, nil
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
