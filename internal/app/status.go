package app

import (
	"context"
	"errors"
	"path/filepath"
	"sort"
	"yeast/internal/project"
	rtm "yeast/internal/runtime"
	"yeast/internal/state"
)

type StatusOptions struct {
	ProjectRoot string
}

type StatusInstanceResult struct {
	Name               string
	Status             string
	PID                int
	ManagementIP       string
	SSHPort            int
	RuntimeDir         string
	ProvisioningStatus string
	LastError          string
}

type StatusResult struct {
	ProjectID string
	Instances []StatusInstanceResult
}

func (s *Service) Status(ctx context.Context, options StatusOptions) (StatusResult, error) {
	root := options.ProjectRoot
	if root == "" {
		root = "."
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return StatusResult{}, err
	}

	metadata, err := project.LoadMetadata(absoluteRoot)
	if err != nil {
		if errors.Is(err, project.ErrMetadataNotFound) {
			return StatusResult{}, WrapError(ErrorCodePrecondition, err.Error(), err)
		}
		return StatusResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	yeastHome, err := s.resolveYeastHome()
	if err != nil {
		return StatusResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	paths, err := project.NewPaths(yeastHome, metadata)
	if err != nil {
		return StatusResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}

	lock, err := state.Acquire(paths.StateLock, state.DefaultLockOptions())
	if err != nil {
		return StatusResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	defer func() { _ = lock.Release() }()

	currentState, err := state.Load(paths.StateFile, metadata.ID)
	if err != nil {
		return StatusResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
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
			return StatusResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
		}
	}

	result := StatusResult{
		ProjectID: metadata.ID,
		Instances: make([]StatusInstanceResult, 0, len(currentState.Instances)),
	}
	for name, instance := range currentState.Instances {
		result.Instances = append(result.Instances, StatusInstanceResult{
			Name:               name,
			Status:             instance.Status,
			PID:                instance.PID,
			ManagementIP:       instance.ManagementIP,
			SSHPort:            instance.SSHPort,
			RuntimeDir:         instance.RuntimeDir,
			ProvisioningStatus: instance.ProvisioningStatus,
			LastError:          instance.LastError,
		})
	}

	sort.Slice(result.Instances, func(i, j int) bool {
		return result.Instances[i].Name < result.Instances[j].Name
	})
	return result, nil
}
