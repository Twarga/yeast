package app

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"yeast/internal/project"
	"yeast/internal/state"
)

type StatusOptions struct {
	ProjectRoot string
}

type StatusInstanceResult struct {
	Name               string                   `json:"name"`
	Status             string                   `json:"status"`
	PID                int                      `json:"pid"`
	ManagementIP       string                   `json:"management_ip,omitempty"`
	SSHPort            int                      `json:"ssh_port,omitempty"`
	User               string                   `json:"user,omitempty"`
	LabIP              string                   `json:"lab_ip,omitempty"`
	RuntimeDir         string                   `json:"runtime_dir,omitempty"`
	ProvisionLogPath   string                   `json:"provision_log_path,omitempty"`
	ProvisioningStatus state.ProvisioningStatus `json:"provisioning_status,omitempty"`
	LastError          string                   `json:"last_error,omitempty"`
}

type StatusResult struct {
	ProjectID string                 `json:"project_id"`
	Instances []StatusInstanceResult `json:"instances"`
}

func (s *Service) Status(ctx context.Context, options StatusOptions) (StatusResult, error) {
	root := options.ProjectRoot
	if root == "" {
		root = "."
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return StatusResult{}, WrapError(ErrorCodeInternal, fmt.Sprintf("resolve project root: %v", err), err)
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

	changed := s.reconcileStateWithRuntime(ctx, &currentState)
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
			User:               instance.User,
			LabIP:              instance.LabIP,
			RuntimeDir:         instance.RuntimeDir,
			ProvisionLogPath:   instance.ProvisionLogPath,
			ProvisioningStatus: instance.ProvisioningStatus,
			LastError:          instance.LastError,
		})
	}

	sort.Slice(result.Instances, func(i, j int) bool {
		return result.Instances[i].Name < result.Instances[j].Name
	})
	return result, nil
}
