package app

import (
	"context"
	"errors"
	"fmt"
	"github.com/Twarga/yeast/internal/config"
	"github.com/Twarga/yeast/internal/project"
	rtm "github.com/Twarga/yeast/internal/runtime"
	"github.com/Twarga/yeast/internal/state"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type CleanOptions struct {
	ProjectRoot string
	Timeout     time.Duration
	Events      EventSink
}

type CleanInstanceResult struct {
	Name        string `json:"name"`
	Status      string `json:"status"`
	CleanedPIDs []int  `json:"cleaned_pids,omitempty"`
}

type CleanResult struct {
	ProjectID string                `json:"project_id"`
	Instances []CleanInstanceResult `json:"instances"`
}

type orphanCleaner interface {
	CleanOrphans(ctx context.Context, targets []rtm.CleanupTarget, timeout time.Duration) ([]rtm.CleanupResult, error)
}

func (s *Service) Clean(ctx context.Context, options CleanOptions) (CleanResult, error) {
	root := options.ProjectRoot
	if root == "" {
		root = "."
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return CleanResult{}, WrapError(ErrorCodeInternal, fmt.Sprintf("resolve project root: %v", err), err)
	}

	metadata, err := project.LoadMetadata(absoluteRoot)
	if err != nil {
		if errors.Is(err, project.ErrMetadataNotFound) {
			return CleanResult{}, WrapError(ErrorCodePrecondition, err.Error(), err)
		}
		return CleanResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	emitEvent(options.Events, "clean", EventProjectLoaded, EventOptions{
		ProjectID: metadata.ID,
		Message:   "Project metadata loaded",
	})

	yeastHome, err := s.resolveYeastHome()
	if err != nil {
		return CleanResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	paths, err := project.NewPaths(yeastHome, metadata)
	if err != nil {
		return CleanResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	if err := os.MkdirAll(paths.ProjectDir, 0755); err != nil {
		return CleanResult{}, WrapError(ErrorCodeInternal, fmt.Sprintf("create project runtime directory %s: %v", paths.ProjectDir, err), err)
	}

	lock, err := state.Acquire(paths.StateLock, state.DefaultLockOptions())
	if err != nil {
		lockErr := WrapError(ErrorCodeInternal, err.Error(), err)
		lockErr = NormalizeError(WithDetails(lockErr, map[string]any{
			"likely_cause": "another Yeast command is using this project or a stale lock file is blocking it",
		}))
		lockErr = NormalizeError(WithRecovery(lockErr,
			"wait for the active Yeast command to finish and retry",
			fmt.Sprintf("remove the stale lock if no Yeast process is running: rm %q", paths.StateLock),
		))
		return CleanResult{}, lockErr
	}
	defer func() { _ = lock.Release() }()

	currentState, err := state.Load(paths.StateFile, metadata.ID)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			currentState = state.New(metadata.ID)
		} else {
			return CleanResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
		}
	}
	_ = addConfiguredRuntimeDirs(&currentState, absoluteRoot, paths)

	timeout := options.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	targets := cleanupTargets(currentState, absoluteRoot, paths)
	cleaned, err := s.cleanOrphanedQEMU(ctx, targets, timeout)
	if err != nil {
		cleanErr := WrapError(ErrorCodeInternal, err.Error(), err)
		cleanErr = NormalizeError(WithDetails(cleanErr, map[string]any{
			"likely_cause": "Yeast could not kill or reap one or more orphaned QEMU processes",
			"cleanup":      "Automatic orphan cleanup did not finish successfully.",
		}))
		cleanErr = NormalizeError(WithRecovery(cleanErr, "yeast clean", "ps -ef | grep qemu"))
		return CleanResult{}, cleanErr
	}
	cleanedPIDs := cleanedPIDsByName(cleaned)

	names := make([]string, 0, len(currentState.Instances))
	for name := range currentState.Instances {
		names = append(names, name)
	}
	sort.Strings(names)

	result := CleanResult{
		ProjectID: metadata.ID,
		Instances: make([]CleanInstanceResult, 0, len(names)),
	}
	for _, name := range names {
		instance := currentState.Instances[name]
		if instance.PID > 0 || instance.RuntimeDir != "" {
			if err := s.runtime.Destroy(ctx, rtm.RuntimeInstance{
				Name:       name,
				RuntimeDir: instance.RuntimeDir,
				PID:        instance.PID,
			}); err != nil {
				return CleanResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
			}
		}
		result.Instances = append(result.Instances, CleanInstanceResult{
			Name:        name,
			Status:      "cleaned",
			CleanedPIDs: cleanedPIDs[name],
		})
		emitEvent(options.Events, "clean", EventInstanceDestroyed, EventOptions{
			ProjectID: metadata.ID,
			Instance:  name,
			Message:   "Instance cleaned",
			Data: map[string]any{
				"status":       "cleaned",
				"cleaned_pids": cleanedPIDs[name],
			},
		})
		delete(currentState.Instances, name)
	}

	if err := state.Save(paths.StateFile, currentState); err != nil {
		return CleanResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	emitEvent(options.Events, "clean", EventWorkflowCompleted, EventOptions{
		ProjectID: metadata.ID,
		Message:   "Workflow completed",
	})
	return result, nil
}

func (s *Service) cleanOrphanedQEMU(ctx context.Context, targets []rtm.CleanupTarget, timeout time.Duration) ([]rtm.CleanupResult, error) {
	cleaner, ok := s.runtime.(orphanCleaner)
	if !ok || len(targets) == 0 {
		return nil, nil
	}
	return cleaner.CleanOrphans(ctx, targets, timeout)
}

func cleanupTargets(currentState state.State, projectRoot string, paths project.Paths) []rtm.CleanupTarget {
	managementHost := defaultManagementHost
	cfg, err := config.Load(filepath.Join(projectRoot, ConfigFileName))
	if err == nil {
		managementHost = cfg.ManagementHost
	}

	targets := make([]rtm.CleanupTarget, 0, len(currentState.Instances))
	for name, instance := range currentState.Instances {
		targets = append(targets, rtm.CleanupTarget{
			Name:       name,
			RuntimeDir: instance.RuntimeDir,
			SSHHost:    managementHost,
			SSHPort:    instance.SSHPort,
		})
	}

	if err != nil {
		return targets
	}
	for _, instance := range cfg.Instances {
		runtimeDir, err := paths.InstanceDir(instance.Name)
		if err != nil {
			continue
		}
		targets = append(targets, rtm.CleanupTarget{
			Name:       instance.Name,
			RuntimeDir: runtimeDir,
			SSHHost:    managementHost,
			SSHPort:    instance.SSHPort,
		})
	}
	return targets
}

func addConfiguredRuntimeDirs(currentState *state.State, projectRoot string, paths project.Paths) error {
	cfg, err := config.Load(filepath.Join(projectRoot, ConfigFileName))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	for _, instance := range cfg.Instances {
		if _, exists := currentState.Instances[instance.Name]; exists {
			continue
		}
		runtimeDir, err := paths.InstanceDir(instance.Name)
		if err != nil {
			continue
		}
		if _, err := os.Stat(runtimeDir); err != nil {
			continue
		}
		currentState.Instances[instance.Name] = state.InstanceState{
			Status:     "unknown",
			RuntimeDir: runtimeDir,
		}
	}
	return nil
}

func cleanedPIDsByName(cleaned []rtm.CleanupResult) map[string][]int {
	grouped := make(map[string][]int)
	for _, result := range cleaned {
		grouped[result.Name] = append(grouped[result.Name], result.PID)
	}
	return grouped
}

// CleanSnapshots deletes every snapshot for every instance — chain-safe —
// then wipes each instance's snapshot directory.
func (s *Service) CleanSnapshots(ctx context.Context, options CleanOptions) (CleanResult, error) {
	root := options.ProjectRoot
	if root == "" {
		root = "."
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return CleanResult{}, WrapError(ErrorCodeInternal, fmt.Sprintf("resolve project root: %v", err), err)
	}
	metadata, err := project.LoadMetadata(absoluteRoot)
	if err != nil {
		if errors.Is(err, project.ErrMetadataNotFound) {
			return CleanResult{}, WrapError(ErrorCodePrecondition, err.Error(), err)
		}
		return CleanResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	yeastHome, err := s.resolveYeastHome()
	if err != nil {
		return CleanResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	paths, err := project.NewPaths(yeastHome, metadata)
	if err != nil {
		return CleanResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	lock, err := state.Acquire(paths.StateLock, state.DefaultLockOptions())
	if err != nil {
		return CleanResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	defer func() { _ = lock.Release() }()
	currentState, err := state.Load(paths.StateFile, metadata.ID)
	if err != nil {
		return CleanResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}

	result := CleanResult{ProjectID: metadata.ID, Instances: []CleanInstanceResult{}}
	for _, name := range sortedInstanceNames(currentState) {
		instance := currentState.Instances[name]
		if instance.Status == "running" {
			return CleanResult{}, WrapError(ErrorCodePrecondition, fmt.Sprintf("instance %q is running — stop it before cleaning snapshots", name), nil)
		}
		deleted := 0
		for snapName, snap := range instance.Snapshots {
			kind := snap.Kind
			if kind == "" {
				kind = "copy"
			}
			if snap.Dir != "" {
				_ = os.Chmod(filepath.Join(snap.Dir, "base.qcow2"), 0644)
			}
			if err := s.runtime.DeleteSnapshot(ctx, rtm.DeleteSnapshotPlan{
				Path: snap.DiskPath, Kind: kind,
				RuntimeDir: instance.RuntimeDir, ActiveDiskPath: instance.ActiveDisk,
			}); err != nil {
				return CleanResult{}, WrapError(ErrorCodeInternal, fmt.Sprintf("delete snapshot %q for %s: %v", snapName, name, err), err)
			}
			deleted++
		}
		if snapRoot := snapDirOf(paths, name); snapRoot != "" {
			_ = os.RemoveAll(snapRoot)
		}
		instance.Snapshots = nil
		currentState.Instances[name] = instance
		result.Instances = append(result.Instances, CleanInstanceResult{
			Name: name, Status: fmt.Sprintf("%d snapshot(s) cleaned", deleted),
		})
		emitEvent(options.Events, "clean", EventInstanceDestroyed, EventOptions{
			ProjectID: metadata.ID, Instance: name,
			Message: fmt.Sprintf("Cleaned %d snapshot(s)", deleted),
		})
	}
	if err := state.Save(paths.StateFile, currentState); err != nil {
		return CleanResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	return result, nil
}
