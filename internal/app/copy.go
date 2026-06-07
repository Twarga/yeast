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
	provssh "yeast/internal/provision/ssh"
	"yeast/internal/state"
)

func (s *Service) Copy(ctx context.Context, options CopyOptions) (CopyResult, error) {
	root := options.ProjectRoot
	if root == "" {
		root = "."
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return CopyResult{}, WrapError(ErrorCodeInternal, fmt.Sprintf("resolve project root: %v", err), err)
	}
	if options.Direction != CopyToGuest && options.Direction != CopyFromGuest {
		return CopyResult{}, WrapError(ErrorCodeInvalidArgument, "copy direction must be to_guest or from_guest", nil)
	}
	if options.Source == "" {
		return CopyResult{}, WrapError(ErrorCodeInvalidArgument, "copy source is required", nil)
	}
	if options.Destination == "" {
		return CopyResult{}, WrapError(ErrorCodeInvalidArgument, "copy destination is required", nil)
	}

	localSource, localDestination, err := validateCopyLocalPaths(absoluteRoot, options)
	if err != nil {
		return CopyResult{}, err
	}

	metadata, err := project.LoadMetadata(absoluteRoot)
	if err != nil {
		if errors.Is(err, project.ErrMetadataNotFound) {
			return CopyResult{}, WrapError(ErrorCodePrecondition, err.Error(), err)
		}
		return CopyResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	yeastHome, err := s.resolveYeastHome()
	if err != nil {
		return CopyResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	paths, err := project.NewPaths(yeastHome, metadata)
	if err != nil {
		return CopyResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}

	lock, err := state.Acquire(paths.StateLock, state.DefaultLockOptions())
	if err != nil {
		return CopyResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	defer func() { _ = lock.Release() }()

	cfg, err := config.Load(filepath.Join(absoluteRoot, ConfigFileName))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return CopyResult{}, WrapError(ErrorCodePrecondition, err.Error(), err)
		}
		return CopyResult{}, WrapError(ErrorCodeInvalidArgument, err.Error(), err)
	}
	currentState, err := state.Load(paths.StateFile, metadata.ID)
	if err != nil {
		return CopyResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	changed, err := s.reconcileProjectState(ctx, &currentState)
	if err != nil {
		return CopyResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	if changed {
		if err := state.Save(paths.StateFile, currentState); err != nil {
			return CopyResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
		}
	}

	selectedName, selectedState, err := selectSSHInstance(currentState, options.Target)
	if err != nil {
		return CopyResult{}, err
	}
	instanceCfg, ok := lookupConfigInstance(cfg, selectedName)
	if !ok {
		return CopyResult{}, WrapError(ErrorCodeNotFound, fmt.Sprintf("instance %q not found in config", selectedName), nil)
	}
	address, err := s.sshAddress(defaultManagementHost, selectedState.SSHPort)
	if err != nil {
		return CopyResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	if err := s.waitForTCP(ctx, guest.ReadinessOptions{
		Address: address,
		Timeout: defaultReadinessTimeout,
	}); err != nil {
		return CopyResult{}, WrapError(ErrorCodePrecondition, fmt.Sprintf("wait for ssh readiness for %s: %v", selectedName, err), err)
	}

	transport := s.provisionTransport
	if transport == nil {
		transport = provssh.NewLocalTransport()
	}

	startedAt := time.Now().UTC()
	switch options.Direction {
	case CopyToGuest:
		err = transport.Upload(ctx, provssh.UploadRequest{
			User:        instanceCfg.User,
			Host:        defaultManagementHost,
			Port:        selectedState.SSHPort,
			Source:      localSource,
			Destination: options.Destination,
			Timeout:     options.Timeout,
		})
	case CopyFromGuest:
		err = transport.Download(ctx, provssh.DownloadRequest{
			User:        instanceCfg.User,
			Host:        defaultManagementHost,
			Port:        selectedState.SSHPort,
			Source:      options.Source,
			Destination: localDestination,
			Timeout:     options.Timeout,
		})
	}
	finishedAt := time.Now().UTC()
	if err != nil {
		return CopyResult{}, WrapError(ErrorCodeGuest, fmt.Sprintf("copy on instance %s: %v", selectedName, err), err)
	}

	result := CopyResult{
		ProjectID:   metadata.ID,
		Instance:    selectedName,
		Direction:   options.Direction,
		Source:      options.Source,
		Destination: options.Destination,
		StartedAt:   startedAt,
		FinishedAt:  finishedAt,
		Duration:    finishedAt.Sub(startedAt),
	}
	if options.Direction == CopyToGuest {
		result.Source = localSource
	}
	if options.Direction == CopyFromGuest {
		result.Destination = localDestination
	}
	return result, nil
}

func validateCopyLocalPaths(projectRoot string, options CopyOptions) (string, string, error) {
	switch options.Direction {
	case CopyToGuest:
		localSource, err := normalizeProjectLocalPath(projectRoot, options.Source)
		if err != nil {
			return "", "", WrapError(ErrorCodeInvalidArgument, fmt.Sprintf("resolve local source: %v", err), err)
		}
		info, err := os.Stat(localSource)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return "", "", WrapError(ErrorCodeInvalidArgument, fmt.Sprintf("local source not found: %s", localSource), err)
			}
			return "", "", WrapError(ErrorCodeInvalidArgument, fmt.Sprintf("stat local source %s: %v", localSource, err), err)
		}
		if info.IsDir() {
			return "", "", WrapError(ErrorCodeInvalidArgument, fmt.Sprintf("local source must be a file: %s", localSource), nil)
		}
		return localSource, "", nil
	case CopyFromGuest:
		localDestination, err := normalizeProjectLocalPath(projectRoot, options.Destination)
		if err != nil {
			return "", "", WrapError(ErrorCodeInvalidArgument, fmt.Sprintf("resolve local destination: %v", err), err)
		}
		parent := filepath.Dir(localDestination)
		info, err := os.Stat(parent)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return "", "", WrapError(ErrorCodeInvalidArgument, fmt.Sprintf("local destination parent not found: %s", parent), err)
			}
			return "", "", WrapError(ErrorCodeInvalidArgument, fmt.Sprintf("stat local destination parent %s: %v", parent, err), err)
		}
		if !info.IsDir() {
			return "", "", WrapError(ErrorCodeInvalidArgument, fmt.Sprintf("local destination parent is not a directory: %s", parent), nil)
		}
		return "", localDestination, nil
	default:
		return "", "", WrapError(ErrorCodeInvalidArgument, "copy direction must be to_guest or from_guest", nil)
	}
}

func normalizeProjectLocalPath(projectRoot, path string) (string, error) {
	cleaned := cleanLocalPath(path)
	if filepath.IsAbs(cleaned) {
		return cleaned, nil
	}
	return filepath.Abs(filepath.Join(projectRoot, cleaned))
}
