package app

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"time"
	"yeast/internal/project"
	rtm "yeast/internal/runtime"
	"yeast/internal/state"
)

type DestroyOptions struct {
	ProjectRoot string
	Timeout     time.Duration
	Events      EventSink
}

type DestroyInstanceResult struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type DestroyResult struct {
	ProjectID string                  `json:"project_id"`
	Instances []DestroyInstanceResult `json:"instances"`
}

func (s *Service) Destroy(ctx context.Context, options DestroyOptions) (DestroyResult, error) {
	root := options.ProjectRoot
	if root == "" {
		root = "."
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return DestroyResult{}, WrapError(ErrorCodeInternal, fmt.Sprintf("resolve project root: %v", err), err)
	}

	metadata, err := project.LoadMetadata(absoluteRoot)
	if err != nil {
		if errors.Is(err, project.ErrMetadataNotFound) {
			return DestroyResult{}, WrapError(ErrorCodePrecondition, err.Error(), err)
		}
		return DestroyResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	emitEvent(options.Events, "destroy", EventProjectLoaded, EventOptions{
		ProjectID: metadata.ID,
		Message:   "Project metadata loaded",
	})
	yeastHome, err := s.resolveYeastHome()
	if err != nil {
		return DestroyResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	paths, err := project.NewPaths(yeastHome, metadata)
	if err != nil {
		return DestroyResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}

	lock, err := state.Acquire(paths.StateLock, state.DefaultLockOptions())
	if err != nil {
		return DestroyResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	defer func() { _ = lock.Release() }()

	currentState, err := state.Load(paths.StateFile, metadata.ID)
	if err != nil {
		return DestroyResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	if err := addConfiguredRuntimeDirs(&currentState, absoluteRoot, paths); err != nil {
		return DestroyResult{}, WrapError(ErrorCodeInvalidArgument, err.Error(), err)
	}

	result := DestroyResult{
		ProjectID: metadata.ID,
		Instances: make([]DestroyInstanceResult, 0, len(currentState.Instances)),
	}

	names := make([]string, 0, len(currentState.Instances))
	for name := range currentState.Instances {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		instance := currentState.Instances[name]
		runtimeInstance := rtm.RuntimeInstance{
			Name:       name,
			RuntimeDir: instance.RuntimeDir,
			PID:        instance.PID,
		}

		if instance.Status == "running" && instance.PID > 0 {
			if err := s.runtime.Destroy(ctx, runtimeInstance); err != nil {
				return DestroyResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
			}
		} else if instance.RuntimeDir != "" {
			if err := s.runtime.Destroy(ctx, runtimeInstance); err != nil {
				return DestroyResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
			}
		}

		delete(currentState.Instances, name)
		result.Instances = append(result.Instances, DestroyInstanceResult{
			Name:   name,
			Status: "destroyed",
		})
		emitEvent(options.Events, "destroy", EventInstanceDestroyed, EventOptions{
			ProjectID: metadata.ID,
			Instance:  name,
			Message:   "Instance destroyed",
			Data: map[string]any{
				"status": "destroyed",
			},
		})
	}

	if err := state.Save(paths.StateFile, currentState); err != nil {
		return DestroyResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	emitEvent(options.Events, "destroy", EventWorkflowCompleted, EventOptions{
		ProjectID: metadata.ID,
		Message:   "Workflow completed",
	})

	return result, nil
}
