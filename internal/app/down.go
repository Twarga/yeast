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

type DownOptions struct {
	ProjectRoot string
	Timeout     time.Duration
	Events      EventSink
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
	emitEvent(options.Events, "down", EventProjectLoaded, EventOptions{
		ProjectID: metadata.ID,
		Message:   "Project metadata loaded",
	})
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
	if err := addConfiguredRuntimeDirs(&currentState, absoluteRoot, paths); err != nil {
		return DownResult{}, WrapError(ErrorCodeInvalidArgument, err.Error(), err)
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
		if instance.PID <= 0 && instance.RuntimeDir == "" {
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
			emitEvent(options.Events, "down", EventInstanceStopped, EventOptions{
				ProjectID: metadata.ID,
				Instance:  name,
				Message:   "Instance already stopped",
				Data: map[string]any{
					"status": "already_stopped",
				},
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
		emitEvent(options.Events, "down", EventInstanceStopped, EventOptions{
			ProjectID: metadata.ID,
			Instance:  name,
			Message:   "Instance stopped",
			Data: map[string]any{
				"status": "stopped",
			},
		})
	}

	if err := state.Save(paths.StateFile, currentState); err != nil {
		return DownResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	emitEvent(options.Events, "down", EventWorkflowCompleted, EventOptions{
		ProjectID: metadata.ID,
		Message:   "Workflow completed",
	})

	return result, nil
}

func addConfiguredRuntimeDirs(currentState *state.State, projectRoot string, paths project.Paths) error {
	if currentState == nil {
		return nil
	}
	if currentState.Instances == nil {
		currentState.Instances = make(map[string]state.InstanceState)
	}

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
			return err
		}
		if _, err := os.Stat(runtimeDir); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return err
		}
		currentState.Instances[instance.Name] = state.InstanceState{
			Status:             "stopped",
			User:               instance.User,
			RuntimeDir:         runtimeDir,
			ProvisioningStatus: state.ProvisioningStatusNotStarted,
		}
	}
	return nil
}
