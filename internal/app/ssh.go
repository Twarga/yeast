package app

import (
	"context"
	"fmt"
	"path/filepath"
	"yeast/internal/config"
	"yeast/internal/guest"
	"yeast/internal/project"
	rtm "yeast/internal/runtime"
	"yeast/internal/state"
)

type SSHOptions struct {
	ProjectRoot string
	Target      string
}

type SSHResult struct {
	InstanceName string
	Address      string
	User         string
	Port         int
}

func (s *Service) SSH(ctx context.Context, options SSHOptions) (SSHResult, error) {
	root := options.ProjectRoot
	if root == "" {
		root = "."
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return SSHResult{}, err
	}

	metadata, err := project.LoadMetadata(absoluteRoot)
	if err != nil {
		return SSHResult{}, err
	}
	yeastHome, err := s.resolveYeastHome()
	if err != nil {
		return SSHResult{}, err
	}
	paths, err := project.NewPaths(yeastHome, metadata)
	if err != nil {
		return SSHResult{}, err
	}

	lock, err := state.Acquire(paths.StateLock, state.DefaultLockOptions())
	if err != nil {
		return SSHResult{}, err
	}
	defer func() { _ = lock.Release() }()

	cfg, err := config.Load(filepath.Join(absoluteRoot, ConfigFileName))
	if err != nil {
		return SSHResult{}, err
	}
	currentState, err := state.Load(paths.StateFile, metadata.ID)
	if err != nil {
		return SSHResult{}, err
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
			return SSHResult{}, err
		}
	}

	selectedName, selectedState, err := selectSSHInstance(currentState, options.Target)
	if err != nil {
		return SSHResult{}, err
	}

	instanceCfg, ok := lookupConfigInstance(cfg, selectedName)
	if !ok {
		return SSHResult{}, WrapError(ErrorCodeNotFound, fmt.Sprintf("instance %q not found in config", selectedName), nil)
	}
	address, err := s.sshAddress(defaultManagementHost, selectedState.SSHPort)
	if err != nil {
		return SSHResult{}, err
	}
	args, err := guest.BuildSSHArgs(instanceCfg.User, defaultManagementHost, selectedState.SSHPort)
	if err != nil {
		return SSHResult{}, err
	}
	if err := s.runSSH(ctx, args); err != nil {
		return SSHResult{}, err
	}

	return SSHResult{
		InstanceName: selectedName,
		Address:      address,
		User:         instanceCfg.User,
		Port:         selectedState.SSHPort,
	}, nil
}

func selectSSHInstance(currentState state.State, target string) (string, state.InstanceState, error) {
	if target != "" {
		instance, ok := currentState.Instances[target]
		if !ok {
			return "", state.InstanceState{}, WrapError(ErrorCodeNotFound, fmt.Sprintf("instance %q not found", target), nil)
		}
		if instance.Status != "running" || instance.SSHPort <= 0 {
			return "", state.InstanceState{}, WrapError(ErrorCodePrecondition, fmt.Sprintf("instance %q is not running", target), nil)
		}
		return target, instance, nil
	}

	var matches []struct {
		name     string
		instance state.InstanceState
	}
	for name, instance := range currentState.Instances {
		if instance.Status == "running" && instance.SSHPort > 0 {
			matches = append(matches, struct {
				name     string
				instance state.InstanceState
			}{name: name, instance: instance})
		}
	}

	switch len(matches) {
	case 0:
		return "", state.InstanceState{}, WrapError(ErrorCodePrecondition, "no running instances", nil)
	case 1:
		return matches[0].name, matches[0].instance, nil
	default:
		return "", state.InstanceState{}, WrapError(ErrorCodeInvalidArgument, "multiple running instances; specify a target", nil)
	}
}

func lookupConfigInstance(cfg *config.Config, name string) (config.Instance, bool) {
	if cfg == nil {
		return config.Instance{}, false
	}
	for _, instance := range cfg.Instances {
		if instance.Name == name {
			return instance, true
		}
	}
	return config.Instance{}, false
}
