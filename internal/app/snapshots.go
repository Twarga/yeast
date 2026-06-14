package app

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"yeast/internal/project"
	"yeast/internal/state"
)

type SnapshotsOptions struct {
	ProjectRoot string
	Target      string
}

type SnapshotsResult struct {
	ProjectID string                `json:"project_id"`
	Instance  string                `json:"instance"`
	Snapshots []state.SnapshotState `json:"snapshots"`
}

type DeleteSnapshotOptions struct {
	ProjectRoot string
	Target      string
	Name        string
	Events      EventSink
}

type DeleteSnapshotResult struct {
	ProjectID string `json:"project_id"`
	Instance  string `json:"instance"`
	Snapshot  string `json:"snapshot"`
}

func (s *Service) Snapshots(ctx context.Context, options SnapshotsOptions) (SnapshotsResult, error) {
	root := options.ProjectRoot
	if root == "" {
		root = "."
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return SnapshotsResult{}, WrapError(ErrorCodeInternal, fmt.Sprintf("resolve project root: %v", err), err)
	}

	metadata, err := project.LoadMetadata(absoluteRoot)
	if err != nil {
		if errors.Is(err, project.ErrMetadataNotFound) {
			return SnapshotsResult{}, WrapError(ErrorCodePrecondition, err.Error(), err)
		}
		return SnapshotsResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	yeastHome, err := s.resolveYeastHome()
	if err != nil {
		return SnapshotsResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	paths, err := project.NewPaths(yeastHome, metadata)
	if err != nil {
		return SnapshotsResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}

	lock, err := state.Acquire(paths.StateLock, state.DefaultLockOptions())
	if err != nil {
		return SnapshotsResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	defer func() { _ = lock.Release() }()

	currentState, err := state.Load(paths.StateFile, metadata.ID)
	if err != nil {
		return SnapshotsResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}

	snapshots, err := listInstanceSnapshots(currentState, options.Target)
	if err != nil {
		return SnapshotsResult{}, err
	}

	return SnapshotsResult{
		ProjectID: metadata.ID,
		Instance:  options.Target,
		Snapshots: snapshots,
	}, nil
}

func (s *Service) DeleteSnapshot(ctx context.Context, options DeleteSnapshotOptions) (DeleteSnapshotResult, error) {
	root := options.ProjectRoot
	if root == "" {
		root = "."
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return DeleteSnapshotResult{}, WrapError(ErrorCodeInternal, fmt.Sprintf("resolve project root: %v", err), err)
	}
	if strings.TrimSpace(options.Target) == "" {
		return DeleteSnapshotResult{}, WrapError(ErrorCodeInvalidArgument, "snapshot target instance is required", nil)
	}
	if !isValidSnapshotName(options.Name) {
		return DeleteSnapshotResult{}, WrapError(ErrorCodeInvalidArgument, fmt.Sprintf("invalid snapshot name %q", options.Name), nil)
	}

	metadata, err := project.LoadMetadata(absoluteRoot)
	if err != nil {
		if errors.Is(err, project.ErrMetadataNotFound) {
			return DeleteSnapshotResult{}, WrapError(ErrorCodePrecondition, err.Error(), err)
		}
		return DeleteSnapshotResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	yeastHome, err := s.resolveYeastHome()
	if err != nil {
		return DeleteSnapshotResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	paths, err := project.NewPaths(yeastHome, metadata)
	if err != nil {
		return DeleteSnapshotResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}

	lock, err := state.Acquire(paths.StateLock, state.DefaultLockOptions())
	if err != nil {
		return DeleteSnapshotResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	defer func() { _ = lock.Release() }()

	currentState, err := state.Load(paths.StateFile, metadata.ID)
	if err != nil {
		return DeleteSnapshotResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}

	instance, ok := currentState.Instances[options.Target]
	if !ok {
		return DeleteSnapshotResult{}, WrapError(ErrorCodeNotFound, fmt.Sprintf("instance %q not found", options.Target), nil)
	}
	snapshot, ok := instance.Snapshots[options.Name]
	if !ok {
		return DeleteSnapshotResult{}, WrapError(ErrorCodeNotFound, fmt.Sprintf("snapshot %q not found for instance %q", options.Name, options.Target), nil)
	}

	if err := s.runtime.DeleteSnapshot(ctx, snapshot.DiskPath); err != nil {
		if strings.Contains(err.Error(), "no such file or directory") {
			return DeleteSnapshotResult{}, WrapError(ErrorCodeNotFound, err.Error(), err)
		}
		return DeleteSnapshotResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}

	delete(instance.Snapshots, options.Name)
	currentState.Instances[options.Target] = instance
	if err := state.Save(paths.StateFile, currentState); err != nil {
		return DeleteSnapshotResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}

	emitEvent(options.Events, "delete-snapshot", EventSnapshotCreated, EventOptions{
		ProjectID: metadata.ID,
		Instance:  options.Target,
		Message:   fmt.Sprintf("Snapshot %s deleted", options.Name),
	})

	return DeleteSnapshotResult{
		ProjectID: metadata.ID,
		Instance:  options.Target,
		Snapshot:  options.Name,
	}, nil
}

func listInstanceSnapshots(currentState state.State, target string) ([]state.SnapshotState, error) {
	if target == "" {
		return nil, WrapError(ErrorCodeInvalidArgument, "snapshot target instance is required", nil)
	}

	instance, ok := currentState.Instances[target]
	if !ok {
		return nil, WrapError(ErrorCodeNotFound, fmt.Sprintf("instance %q not found", target), nil)
	}

	return state.SortedSnapshots(instance), nil
}
