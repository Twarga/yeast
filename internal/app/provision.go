package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
	"yeast/internal/config"
	"yeast/internal/guest"
	"yeast/internal/project"
	"yeast/internal/provision"
	rtm "yeast/internal/runtime"
	"yeast/internal/state"
)

type ProvisionOptions struct {
	ProjectRoot      string
	Target           string
	ReadinessTimeout time.Duration
}

type ProvisionInstanceResult struct {
	Name               string
	ProvisioningStatus state.ProvisioningStatus
	SSHAddress         string
	SSHPort            int
	ProvisionLogPath   string
	LastError          string
}

type ProvisionResult struct {
	ProjectID string
	Instance  ProvisionInstanceResult
}

func (s *Service) Provision(ctx context.Context, options ProvisionOptions) (ProvisionResult, error) {
	root := options.ProjectRoot
	if root == "" {
		root = "."
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return ProvisionResult{}, WrapError(ErrorCodeInternal, fmt.Sprintf("resolve project root: %v", err), err)
	}

	metadata, err := project.LoadMetadata(absoluteRoot)
	if err != nil {
		if errors.Is(err, project.ErrMetadataNotFound) {
			return ProvisionResult{}, WrapError(ErrorCodePrecondition, err.Error(), err)
		}
		return ProvisionResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}

	yeastHome, err := s.resolveYeastHome()
	if err != nil {
		return ProvisionResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	paths, err := project.NewPaths(yeastHome, metadata)
	if err != nil {
		return ProvisionResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}

	lock, err := state.Acquire(paths.StateLock, state.DefaultLockOptions())
	if err != nil {
		return ProvisionResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	defer func() { _ = lock.Release() }()

	cfg, err := config.Load(filepath.Join(absoluteRoot, ConfigFileName))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ProvisionResult{}, WrapError(ErrorCodePrecondition, err.Error(), err)
		}
		return ProvisionResult{}, WrapError(ErrorCodeInvalidArgument, err.Error(), err)
	}
	currentState, err := state.Load(paths.StateFile, metadata.ID)
	if err != nil {
		return ProvisionResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
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
			return ProvisionResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
		}
	}

	selectedName, selectedState, err := selectSSHInstance(currentState, options.Target)
	if err != nil {
		return ProvisionResult{}, err
	}
	instanceCfg, ok := lookupConfigInstance(cfg, selectedName)
	if !ok {
		return ProvisionResult{}, WrapError(ErrorCodeNotFound, fmt.Sprintf("instance %q not found in config", selectedName), nil)
	}
	address, err := s.sshAddress(defaultManagementHost, selectedState.SSHPort)
	if err != nil {
		return ProvisionResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}

	readinessTimeout := options.ReadinessTimeout
	if readinessTimeout <= 0 {
		readinessTimeout = defaultReadinessTimeout
	}
	if err := s.waitForTCP(ctx, guest.ReadinessOptions{
		Address: address,
		Timeout: readinessTimeout,
	}); err != nil {
		return ProvisionResult{}, WrapError(ErrorCodePrecondition, fmt.Sprintf("wait for ssh readiness for %s: %v", selectedName, err), err)
	}

	provisionPlan, err := resolveProvisionPlan(absoluteRoot, provision.BuildPlan(instanceCfg, cfg.Provision))
	if err != nil {
		return ProvisionResult{}, WrapError(ErrorCodeInvalidArgument, fmt.Sprintf("resolve provision plan for %s: %v", selectedName, err), err)
	}

	instanceState := selectedState
	if instanceState.ProvisionLogPath == "" {
		runtimeDir := instanceState.RuntimeDir
		if runtimeDir == "" {
			runtimeDir, err = paths.InstanceDir(selectedName)
			if err != nil {
				return ProvisionResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
			}
			instanceState.RuntimeDir = runtimeDir
		}
		instanceState.ProvisionLogPath = filepath.Join(runtimeDir, "provision.log")
	}
	if provisionPlan.Empty() {
		instanceState.ProvisioningStatus = state.ProvisioningStatusReady
	} else {
		instanceState.ProvisioningStatus = state.ProvisioningStatusRunning
	}
	instanceState.LastError = ""
	currentState.Instances[selectedName] = instanceState
	if err := state.Save(paths.StateFile, currentState); err != nil {
		return ProvisionResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}

	provisionResult, err := s.runProvisionPlan(ctx, instanceCfg, selectedState.SSHPort, instanceState.ProvisionLogPath, provisionPlan)
	instanceState.ProvisioningStatus = provisionResult.Status
	instanceState.LastError = ""
	if err != nil {
		instanceState.ProvisioningStatus = state.ProvisioningStatusFailed
		instanceState.LastError = err.Error()
		currentState.Instances[selectedName] = instanceState
		if saveErr := state.Save(paths.StateFile, currentState); saveErr != nil {
			return ProvisionResult{}, WrapError(ErrorCodeInternal, saveErr.Error(), saveErr)
		}
		appErr := NormalizeError(err)
		if appErr.Code == ErrorCodeInternal {
			return ProvisionResult{}, appErr
		}
		return ProvisionResult{}, WrapError(ErrorCodePrecondition, fmt.Sprintf("provision instance %s: %v", selectedName, err), err)
	}

	currentState.Instances[selectedName] = instanceState
	if err := state.Save(paths.StateFile, currentState); err != nil {
		return ProvisionResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}

	return ProvisionResult{
		ProjectID: metadata.ID,
		Instance: ProvisionInstanceResult{
			Name:               selectedName,
			ProvisioningStatus: instanceState.ProvisioningStatus,
			SSHAddress:         address,
			SSHPort:            selectedState.SSHPort,
			ProvisionLogPath:   instanceState.ProvisionLogPath,
			LastError:          instanceState.LastError,
		},
	}, nil
}
