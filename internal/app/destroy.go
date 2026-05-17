package app

import (
	"context"
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
}

type DestroyInstanceResult struct {
	Name   string
	Status string
}

type DestroyResult struct {
	ProjectID string
	Instances []DestroyInstanceResult
}

func (s *Service) Destroy(ctx context.Context, options DestroyOptions) (DestroyResult, error) {
	root := options.ProjectRoot
	if root == "" {
		root = "."
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return DestroyResult{}, err
	}

	metadata, err := project.LoadMetadata(absoluteRoot)
	if err != nil {
		return DestroyResult{}, err
	}
	yeastHome, err := s.resolveYeastHome()
	if err != nil {
		return DestroyResult{}, err
	}
	paths, err := project.NewPaths(yeastHome, metadata)
	if err != nil {
		return DestroyResult{}, err
	}

	lock, err := state.Acquire(paths.StateLock, state.DefaultLockOptions())
	if err != nil {
		return DestroyResult{}, err
	}
	defer func() { _ = lock.Release() }()

	currentState, err := state.Load(paths.StateFile, metadata.ID)
	if err != nil {
		return DestroyResult{}, err
	}

	timeout := options.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
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
	}

	if err := state.Save(paths.StateFile, currentState); err != nil {
		return DestroyResult{}, err
	}

	return result, nil
}
