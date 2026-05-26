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

type DownOptions struct {
	ProjectRoot string
	Timeout     time.Duration
}

type DownInstanceResult struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type DownResult struct {
	ProjectID string               `json:"project_id"`
	Instances []DownInstanceResult `json:"instances"`
}

func (s *Service) Down(ctx context.Context, options DownOptions) (DownResult, error) {
	root := options.ProjectRoot
	if root == "" {
		root = "."
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return DownResult{}, WrapError(ErrorCodeInternal, fmt.Sprintf("resolve project root: %v", err), err)
	}

	metadata, err := project.LoadMetadata(absoluteRoot)
	if err != nil {
		if errors.Is(err, project.ErrMetadataNotFound) {
			return DownResult{}, WrapError(ErrorCodePrecondition, err.Error(), err)
		}
		return DownResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	yeastHome, err := s.resolveYeastHome()
	if err != nil {
		return DownResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	paths, err := project.NewPaths(yeastHome, metadata)
	if err != nil {
		return DownResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}

	lock, err := state.Acquire(paths.StateLock, state.DefaultLockOptions())
	if err != nil {
		return DownResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	defer func() { _ = lock.Release() }()

	currentState, err := state.Load(paths.StateFile, metadata.ID)
	if err != nil {
		return DownResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}

	timeout := options.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	result := DownResult{
		ProjectID: metadata.ID,
		Instances: make([]DownInstanceResult, 0, len(currentState.Instances)),
	}

	names := make([]string, 0, len(currentState.Instances))
	for name := range currentState.Instances {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		instance := currentState.Instances[name]
		if instance.Status != "running" || instance.PID <= 0 {
			instance.Status = "stopped"
			instance.PID = 0
			instance.ManagementIP = ""
			instance.SSHPort = 0
			instance.ProvisioningStatus = state.ProvisioningStatusNotStarted
			currentState.Instances[name] = instance
			result.Instances = append(result.Instances, DownInstanceResult{
				Name:   name,
				Status: "already_stopped",
			})
			continue
		}

		err := s.runtime.Stop(ctx, rtm.RuntimeInstance{
			Name:       name,
			RuntimeDir: instance.RuntimeDir,
			PID:        instance.PID,
		}, timeout)
		if err != nil {
			return DownResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
		}
		if err := s.waitForManagementPortRelease(ctx, instance.SSHPort, timeout); err != nil {
			return DownResult{}, WrapError(
				ErrorCodeInternal,
				fmt.Sprintf("wait for ssh_port %d to become available: %v", instance.SSHPort, err),
				err,
			)
		}

		instance.Status = "stopped"
		instance.PID = 0
		instance.ManagementIP = ""
		instance.SSHPort = 0
		instance.ProvisioningStatus = state.ProvisioningStatusNotStarted
		instance.LastError = ""
		currentState.Instances[name] = instance
		result.Instances = append(result.Instances, DownInstanceResult{
			Name:   name,
			Status: "stopped",
		})
	}

	if err := state.Save(paths.StateFile, currentState); err != nil {
		return DownResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}

	return result, nil
}
