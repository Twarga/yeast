package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
	"yeast/internal/config"
	"yeast/internal/project"
	rtm "yeast/internal/runtime"
	"yeast/internal/state"
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

	lock, err := state.Acquire(paths.StateLock, state.DefaultLockOptions())
	if err != nil {
		return CleanResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	defer func() { _ = lock.Release() }()

	currentState, err := state.Load(paths.StateFile, metadata.ID)
	if err != nil {
		return CleanResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	if err := addConfiguredRuntimeDirs(&currentState, absoluteRoot, paths); err != nil {
		return CleanResult{}, WrapError(ErrorCodeInvalidArgument, err.Error(), err)
	}

	timeout := options.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	targets := cleanupTargets(currentState, absoluteRoot, paths)
	cleaned, err := s.cleanOrphanedQEMU(ctx, targets, timeout)
	if err != nil {
		return CleanResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
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
	targets := make([]rtm.CleanupTarget, 0, len(currentState.Instances))
	for name, instance := range currentState.Instances {
		targets = append(targets, rtm.CleanupTarget{
			Name:       name,
			RuntimeDir: instance.RuntimeDir,
			SSHHost:    defaultManagementHost,
			SSHPort:    instance.SSHPort,
		})
	}

	cfg, err := config.Load(filepath.Join(projectRoot, ConfigFileName))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return targets
		}
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
			SSHHost:    defaultManagementHost,
			SSHPort:    instance.SSHPort,
		})
	}
	return targets
}

func cleanedPIDsByName(cleaned []rtm.CleanupResult) map[string][]int {
	grouped := make(map[string][]int)
	for _, result := range cleaned {
		grouped[result.Name] = append(grouped[result.Name], result.PID)
	}
	return grouped
}
