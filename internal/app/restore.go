package app

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"yeast/internal/project"
	rtm "yeast/internal/runtime"
	"yeast/internal/state"
)

type RestoreOptions struct {
	ProjectRoot string
	Target      string
	Name        string
	Events      EventSink
}

type RestoreResult struct {
	ProjectID string              `json:"project_id"`
	Instance  string              `json:"instance"`
	Snapshot  state.SnapshotState `json:"snapshot"`
}

func (s *Service) Restore(ctx context.Context, options RestoreOptions) (RestoreResult, error) {
	root := options.ProjectRoot
	if root == "" {
		root = "."
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return RestoreResult{}, WrapError(ErrorCodeInternal, fmt.Sprintf("resolve project root: %v", err), err)
	}
	if strings.TrimSpace(options.Target) == "" {
		return RestoreResult{}, WrapError(ErrorCodeInvalidArgument, "restore target instance is required", nil)
	}
	if !isValidSnapshotName(options.Name) {
		return RestoreResult{}, WrapError(ErrorCodeInvalidArgument, fmt.Sprintf("invalid snapshot name %q", options.Name), nil)
	}

	metadata, err := project.LoadMetadata(absoluteRoot)
	if err != nil {
		if errors.Is(err, project.ErrMetadataNotFound) {
			return RestoreResult{}, WrapError(ErrorCodePrecondition, err.Error(), err)
		}
		return RestoreResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	emitEvent(options.Events, "restore", EventProjectLoaded, EventOptions{
		ProjectID: metadata.ID,
		Message:   "Project metadata loaded",
	})
	yeastHome, err := s.resolveYeastHome()
	if err != nil {
		return RestoreResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	paths, err := project.NewPaths(yeastHome, metadata)
	if err != nil {
		return RestoreResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}

	lock, err := state.Acquire(paths.StateLock, state.DefaultLockOptions())
	if err != nil {
		return RestoreResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	defer func() { _ = lock.Release() }()

	currentState, err := state.Load(paths.StateFile, metadata.ID)
	if err != nil {
		return RestoreResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}

	instance, ok := currentState.Instances[options.Target]
	if !ok {
		return RestoreResult{}, WrapError(ErrorCodeNotFound, fmt.Sprintf("instance %q not found", options.Target), nil)
	}
	if instance.Status != "stopped" {
		return RestoreResult{}, WrapError(ErrorCodePrecondition, fmt.Sprintf("instance %q must be stopped before restore", options.Target), nil)
	}
	if instance.RuntimeDir == "" {
		return RestoreResult{}, WrapError(ErrorCodeInternal, fmt.Sprintf("instance %q runtime directory is missing", options.Target), nil)
	}
	snapshot, ok := instance.Snapshots[options.Name]
	if !ok {
		return RestoreResult{}, WrapError(ErrorCodeNotFound, fmt.Sprintf("snapshot %q not found for instance %q", options.Name, options.Target), nil)
	}

	diskPath := filepath.Join(instance.RuntimeDir, "disk.qcow2")
	emitEvent(options.Events, "restore", EventRestoreStarted, EventOptions{
		ProjectID: metadata.ID,
		Instance:  options.Target,
		Message:   "Restore started",
		Data: map[string]any{
			"snapshot":  options.Name,
			"disk_path": diskPath,
		},
	})
	if err := s.runtime.RestoreSnapshot(ctx, rtm.SnapshotPlan{
		InstanceDiskPath: diskPath,
		SnapshotPath:     snapshot.DiskPath,
	}); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return RestoreResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
		}
		if strings.Contains(err.Error(), "inspect snapshot file") {
			return RestoreResult{}, WrapError(ErrorCodeNotFound, err.Error(), err)
		}
		return RestoreResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	instance.ProvisioningStatus = state.ProvisioningStatusReady
	currentState.Instances[options.Target] = instance
	if err := state.Save(paths.StateFile, currentState); err != nil {
		return RestoreResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}

	emitEvent(options.Events, "restore", EventRestoreFinished, EventOptions{
		ProjectID: metadata.ID,
		Instance:  options.Target,
		Message:   "Restore finished",
		Data: map[string]any{
			"snapshot": options.Name,
		},
	})
	emitEvent(options.Events, "restore", EventWorkflowCompleted, EventOptions{
		ProjectID: metadata.ID,
		Instance:  options.Target,
		Message:   "Workflow completed",
	})

	return RestoreResult{
		ProjectID: metadata.ID,
		Instance:  options.Target,
		Snapshot:  snapshot,
	}, nil
}
