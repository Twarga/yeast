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

type DownOptions struct {
	ProjectRoot string
	Timeout     time.Duration
}

type DownInstanceResult struct {
	Name   string
	Status string
}

type DownResult struct {
	ProjectID string
	Instances []DownInstanceResult
}

func (s *Service) Down(ctx context.Context, options DownOptions) (DownResult, error) {
	root := options.ProjectRoot
	if root == "" {
		root = "."
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return DownResult{}, err
	}

	metadata, err := project.LoadMetadata(absoluteRoot)
	if err != nil {
		return DownResult{}, err
	}
	yeastHome, err := s.resolveYeastHome()
	if err != nil {
		return DownResult{}, err
	}
	paths, err := project.NewPaths(yeastHome, metadata)
	if err != nil {
		return DownResult{}, err
	}

	lock, err := state.Acquire(paths.StateLock, state.DefaultLockOptions())
	if err != nil {
		return DownResult{}, err
	}
	defer func() { _ = lock.Release() }()

	currentState, err := state.Load(paths.StateFile, metadata.ID)
	if err != nil {
		return DownResult{}, err
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
			instance.ProvisioningStatus = ""
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
			return DownResult{}, err
		}

		instance.Status = "stopped"
		instance.PID = 0
		instance.ManagementIP = ""
		instance.SSHPort = 0
		instance.ProvisioningStatus = ""
		instance.LastError = ""
		currentState.Instances[name] = instance
		result.Instances = append(result.Instances, DownInstanceResult{
			Name:   name,
			Status: "stopped",
		})
	}

	if err := state.Save(paths.StateFile, currentState); err != nil {
		return DownResult{}, err
	}

	return result, nil
}
