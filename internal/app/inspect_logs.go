package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"yeast/internal/project"
	rtm "yeast/internal/runtime"
	"yeast/internal/state"
)

func (s *Service) Inspect(ctx context.Context, options InspectOptions) (InspectResult, error) {
	root := options.ProjectRoot
	if root == "" {
		root = "."
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return InspectResult{}, WrapError(ErrorCodeInternal, fmt.Sprintf("resolve project root: %v", err), err)
	}
	if strings.TrimSpace(options.Target) == "" {
		return InspectResult{}, WrapError(ErrorCodeInvalidArgument, "inspect target instance is required", nil)
	}

	metadata, currentState, err := s.loadProjectStateForGuestControl(ctx, absoluteRoot)
	if err != nil {
		return InspectResult{}, err
	}

	instance, ok := currentState.Instances[options.Target]
	if !ok {
		return InspectResult{}, WrapError(ErrorCodeNotFound, fmt.Sprintf("instance %q not found", options.Target), nil)
	}
	snapshots := state.SortedSnapshots(instance)
	snapshotNames := make([]string, 0, len(snapshots))
	for _, snapshot := range snapshots {
		snapshotNames = append(snapshotNames, snapshot.Name)
	}

	return InspectResult{
		ProjectID: metadata.ID,
		Instance: StatusInstanceResult{
			Name:               options.Target,
			Status:             instance.Status,
			PID:                instance.PID,
			ManagementIP:       instance.ManagementIP,
			SSHPort:            instance.SSHPort,
			LabIP:              instance.LabIP,
			RuntimeDir:         instance.RuntimeDir,
			ProvisionLogPath:   instance.ProvisionLogPath,
			ProvisioningStatus: instance.ProvisioningStatus,
			LastError:          instance.LastError,
		},
		SnapshotNames: snapshotNames,
		SnapshotCount: len(snapshotNames),
	}, nil
}

func (s *Service) Logs(ctx context.Context, options LogsOptions) (LogsResult, error) {
	root := options.ProjectRoot
	if root == "" {
		root = "."
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return LogsResult{}, WrapError(ErrorCodeInternal, fmt.Sprintf("resolve project root: %v", err), err)
	}
	if strings.TrimSpace(options.Target) == "" {
		return LogsResult{}, WrapError(ErrorCodeInvalidArgument, "logs target instance is required", nil)
	}

	metadata, currentState, err := s.loadProjectStateForGuestControl(ctx, absoluteRoot)
	if err != nil {
		return LogsResult{}, err
	}

	instance, ok := currentState.Instances[options.Target]
	if !ok {
		return LogsResult{}, WrapError(ErrorCodeNotFound, fmt.Sprintf("instance %q not found", options.Target), nil)
	}
	if instance.RuntimeDir == "" {
		return LogsResult{}, WrapError(ErrorCodeInternal, fmt.Sprintf("instance %q runtime directory is missing", options.Target), nil)
	}

	logPath := filepath.Join(instance.RuntimeDir, "vm.log")
	content, err := os.ReadFile(logPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return LogsResult{}, WrapError(ErrorCodeNotFound, fmt.Sprintf("vm log not found: %s", logPath), err)
		}
		return LogsResult{}, WrapError(ErrorCodeInternal, fmt.Sprintf("read vm log %s: %v", logPath, err), err)
	}

	return LogsResult{
		ProjectID: metadata.ID,
		Instance:  options.Target,
		LogPath:   logPath,
		Content:   tailLogContent(string(content), options.TailLines),
	}, nil
}

func (s *Service) loadProjectStateForGuestControl(ctx context.Context, absoluteRoot string) (project.Metadata, state.State, error) {
	metadata, err := project.LoadMetadata(absoluteRoot)
	if err != nil {
		if errors.Is(err, project.ErrMetadataNotFound) {
			return project.Metadata{}, state.State{}, WrapError(ErrorCodePrecondition, err.Error(), err)
		}
		return project.Metadata{}, state.State{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	yeastHome, err := s.resolveYeastHome()
	if err != nil {
		return project.Metadata{}, state.State{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	paths, err := project.NewPaths(yeastHome, metadata)
	if err != nil {
		return project.Metadata{}, state.State{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}

	lock, err := state.Acquire(paths.StateLock, state.DefaultLockOptions())
	if err != nil {
		return project.Metadata{}, state.State{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	defer func() { _ = lock.Release() }()

	currentState, err := state.Load(paths.StateFile, metadata.ID)
	if err != nil {
		return project.Metadata{}, state.State{}, WrapError(ErrorCodeInternal, err.Error(), err)
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
			return project.Metadata{}, state.State{}, WrapError(ErrorCodeInternal, err.Error(), err)
		}
	}

	return metadata, currentState, nil
}

func tailLogContent(content string, tailLines int) string {
	if tailLines <= 0 {
		return content
	}
	trimmed := strings.TrimSuffix(content, "\n")
	lines := strings.Split(trimmed, "\n")
	if len(lines) == 1 && lines[0] == "" {
		return ""
	}
	if tailLines >= len(lines) {
		return strings.Join(lines, "\n")
	}
	return strings.Join(lines[len(lines)-tailLines:], "\n")
}
