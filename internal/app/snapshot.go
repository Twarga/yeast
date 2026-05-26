package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"yeast/internal/project"
	rtm "yeast/internal/runtime"
	"yeast/internal/state"
)

type SnapshotOptions struct {
	ProjectRoot string
	Target      string
	Name        string
	Description string
}

type SnapshotResult struct {
	ProjectID string              `json:"project_id"`
	Instance  string              `json:"instance"`
	Snapshot  state.SnapshotState `json:"snapshot"`
}

func (s *Service) Snapshot(ctx context.Context, options SnapshotOptions) (SnapshotResult, error) {
	root := options.ProjectRoot
	if root == "" {
		root = "."
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return SnapshotResult{}, WrapError(ErrorCodeInternal, fmt.Sprintf("resolve project root: %v", err), err)
	}
	if strings.TrimSpace(options.Target) == "" {
		return SnapshotResult{}, WrapError(ErrorCodeInvalidArgument, "snapshot target instance is required", nil)
	}
	if !isValidSnapshotName(options.Name) {
		return SnapshotResult{}, WrapError(ErrorCodeInvalidArgument, fmt.Sprintf("invalid snapshot name %q", options.Name), nil)
	}

	metadata, err := project.LoadMetadata(absoluteRoot)
	if err != nil {
		if errors.Is(err, project.ErrMetadataNotFound) {
			return SnapshotResult{}, WrapError(ErrorCodePrecondition, err.Error(), err)
		}
		return SnapshotResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	yeastHome, err := s.resolveYeastHome()
	if err != nil {
		return SnapshotResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	paths, err := project.NewPaths(yeastHome, metadata)
	if err != nil {
		return SnapshotResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}

	lock, err := state.Acquire(paths.StateLock, state.DefaultLockOptions())
	if err != nil {
		return SnapshotResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	defer func() { _ = lock.Release() }()

	currentState, err := state.Load(paths.StateFile, metadata.ID)
	if err != nil {
		return SnapshotResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}

	instance, ok := currentState.Instances[options.Target]
	if !ok {
		return SnapshotResult{}, WrapError(ErrorCodeNotFound, fmt.Sprintf("instance %q not found", options.Target), nil)
	}
	if instance.Status != "stopped" {
		return SnapshotResult{}, WrapError(ErrorCodePrecondition, fmt.Sprintf("instance %q must be stopped before snapshot", options.Target), nil)
	}
	if instance.RuntimeDir == "" {
		return SnapshotResult{}, WrapError(ErrorCodeInternal, fmt.Sprintf("instance %q runtime directory is missing", options.Target), nil)
	}
	if _, exists := instance.Snapshots[options.Name]; exists {
		return SnapshotResult{}, WrapError(ErrorCodeConflict, fmt.Sprintf("snapshot %q already exists for instance %q", options.Name, options.Target), nil)
	}

	snapshotDir, err := paths.SnapshotDir(options.Target)
	if err != nil {
		return SnapshotResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	snapshotPath := filepath.Join(snapshotDir, options.Name+".qcow2")
	diskPath := filepath.Join(instance.RuntimeDir, "disk.qcow2")
	if err := s.runtime.CreateSnapshot(ctx, rtm.SnapshotPlan{
		InstanceDiskPath: diskPath,
		SnapshotPath:     snapshotPath,
	}); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return SnapshotResult{}, WrapError(ErrorCodeConflict, err.Error(), err)
		}
		return SnapshotResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}

	snapshot := state.SnapshotState{
		Name:           options.Name,
		CreatedAt:      time.Now().UTC(),
		Description:    strings.TrimSpace(options.Description),
		DiskPath:       snapshotPath,
		SourceDiskSize: snapshotSourceDiskSize(diskPath),
	}
	if instance.Snapshots == nil {
		instance.Snapshots = make(map[string]state.SnapshotState)
	}
	instance.Snapshots[options.Name] = snapshot
	currentState.Instances[options.Target] = instance
	if err := state.Save(paths.StateFile, currentState); err != nil {
		return SnapshotResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}

	return SnapshotResult{
		ProjectID: metadata.ID,
		Instance:  options.Target,
		Snapshot:  snapshot,
	}, nil
}

func isValidSnapshotName(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	if name == "." || name == ".." {
		return false
	}
	if strings.ContainsRune(name, filepath.Separator) {
		return false
	}
	return !strings.Contains(name, "..")
}

func snapshotSourceDiskSize(diskPath string) string {
	info, err := os.Stat(diskPath)
	if err != nil || info.Size() <= 0 {
		return ""
	}
	return fmt.Sprintf("%d", info.Size())
}
