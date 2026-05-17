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
	"yeast/internal/guest"
	"yeast/internal/images"
	"yeast/internal/project"
	"yeast/internal/provision/cloudinit"
	rtm "yeast/internal/runtime"
	"yeast/internal/state"
)

const (
	defaultReadinessTimeout = 2 * time.Minute
	defaultManagementHost   = "127.0.0.1"
	firstManagementSSHPort  = 2222
)

type UpOptions struct {
	ProjectRoot      string
	ReadinessTimeout time.Duration
}

type UpInstanceResult struct {
	Name       string
	Status     string
	SSHAddress string
	SSHPort    int
}

type UpResult struct {
	ProjectID string
	Instances []UpInstanceResult
}

func (s *Service) Up(ctx context.Context, options UpOptions) (UpResult, error) {
	root := options.ProjectRoot
	if root == "" {
		root = "."
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return UpResult{}, WrapError(ErrorCodeInternal, fmt.Sprintf("resolve project root: %v", err), err)
	}

	metadata, err := project.LoadMetadata(absoluteRoot)
	if err != nil {
		if errors.Is(err, project.ErrMetadataNotFound) {
			return UpResult{}, WrapError(ErrorCodePrecondition, err.Error(), err)
		}
		return UpResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}

	yeastHome, err := s.resolveYeastHome()
	if err != nil {
		return UpResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	paths, err := project.NewPaths(yeastHome, metadata)
	if err != nil {
		return UpResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	if err := os.MkdirAll(paths.ProjectDir, 0755); err != nil {
		return UpResult{}, WrapError(ErrorCodeInternal, fmt.Sprintf("create project runtime directory %s: %v", paths.ProjectDir, err), err)
	}

	lock, err := state.Acquire(paths.StateLock, state.DefaultLockOptions())
	if err != nil {
		return UpResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	defer func() { _ = lock.Release() }()

	cfg, err := config.Load(filepath.Join(absoluteRoot, ConfigFileName))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return UpResult{}, WrapError(ErrorCodePrecondition, err.Error(), err)
		}
		return UpResult{}, WrapError(ErrorCodeInvalidArgument, err.Error(), err)
	}

	currentState, err := state.Load(paths.StateFile, metadata.ID)
	if err != nil {
		return UpResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	if state.Reconcile(&currentState, state.ReconcileOptions{}) {
		if err := state.Save(paths.StateFile, currentState); err != nil {
			return UpResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
		}
	}

	readinessTimeout := options.ReadinessTimeout
	if readinessTimeout <= 0 {
		readinessTimeout = defaultReadinessTimeout
	}

	result := UpResult{
		ProjectID: metadata.ID,
		Instances: make([]UpInstanceResult, 0, len(cfg.Instances)),
	}

	allocatedPorts := usedManagementPorts(currentState)
	startedInstances := make([]rtm.RuntimeInstance, 0)
	for _, instance := range cfg.Instances {
		if existing, ok := currentState.Instances[instance.Name]; ok && existing.Status == "running" && existing.PID > 0 && existing.SSHPort > 0 {
			address, err := s.sshAddress(defaultManagementHost, existing.SSHPort)
			if err != nil {
				return UpResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
			}
			result.Instances = append(result.Instances, UpInstanceResult{
				Name:       instance.Name,
				Status:     existing.Status,
				SSHAddress: address,
				SSHPort:    existing.SSHPort,
			})
			allocatedPorts[existing.SSHPort] = true
			continue
		}

		image, ok := images.Lookup(instance.Image)
		if !ok {
			return UpResult{}, WrapError(ErrorCodeInvalidArgument, fmt.Sprintf("unsupported image %q", instance.Image), nil)
		}
		cachePaths, err := images.ResolveCachePaths(paths.ImageCache, image.Name)
		if err != nil {
			return UpResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
		}
		if _, err := os.Stat(cachePaths.ImageFile); err != nil {
			message := fmt.Sprintf("image %s not found in cache at %s: run `yeast pull %s`", image.Name, cachePaths.ImageFile, image.Name)
			return UpResult{}, WrapError(ErrorCodeNotFound, message, err)
		}

		runtimeDir, err := paths.InstanceDir(instance.Name)
		if err != nil {
			return UpResult{}, WrapError(ErrorCodeInvalidArgument, err.Error(), err)
		}
		sshPort, err := chooseManagementSSHPort(currentState, instance, allocatedPorts)
		if err != nil {
			return UpResult{}, WrapError(ErrorCodeInvalidArgument, err.Error(), err)
		}
		allocatedPorts[sshPort] = true

		userKey, err := s.discoverSSHKey()
		if err != nil {
			if errors.Is(err, cloudinit.ErrNoSSHPublicKey) {
				return UpResult{}, WrapError(ErrorCodePrecondition, err.Error(), err)
			}
			return UpResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
		}
		userData, err := s.renderUserData(cloudinit.UserDataInput{
			Hostname:      instance.Hostname,
			Instance:      instance,
			AuthorizedKey: userKey,
		})
		if err != nil {
			return UpResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
		}
		metaData, err := s.renderMetaData(cloudinit.MetaDataInput{Hostname: instance.Hostname})
		if err != nil {
			return UpResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
		}
		seedResult, err := s.createSeedISO(ctx, cloudinit.SeedInput{
			InstanceName: instance.Name,
			RuntimeDir:   runtimeDir,
			UserData:     userData,
			MetaData:     metaData,
		})
		if err != nil {
			return UpResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
		}

		plan := rtm.MachinePlan{
			Name:          instance.Name,
			RuntimeDir:    runtimeDir,
			LogPath:       filepath.Join(runtimeDir, "vm.log"),
			MemoryMiB:     instance.Memory,
			CPUs:          instance.CPUs,
			SeedImagePath: seedResult.ISOPath,
			Disk: rtm.DiskPlan{
				BaseImagePath: cachePaths.ImageFile,
				DiskPath:      filepath.Join(runtimeDir, "disk.qcow2"),
				Size:          instance.DiskSize,
			},
			ManagementNetwork: rtm.NetworkOptions{
				ManagementSSHPort: sshPort,
			},
		}

		if _, err := s.runtime.PrepareDisk(ctx, plan); err != nil {
			return UpResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
		}

		started, err := s.runtime.Start(ctx, plan)
		if err != nil {
			return UpResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
		}
		startedInstances = append(startedInstances, started)

		address, err := s.sshAddress(defaultManagementHost, sshPort)
		if err != nil {
			_ = s.runtime.Stop(ctx, started, 5*time.Second)
			return UpResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
		}
		if err := s.waitForTCP(ctx, guest.ReadinessOptions{
			Address: address,
			Timeout: readinessTimeout,
		}); err != nil {
			_ = s.runtime.Stop(ctx, started, 5*time.Second)
			message := fmt.Sprintf("wait for ssh readiness for %s: %v", instance.Name, err)
			return UpResult{}, WrapError(ErrorCodePrecondition, message, err)
		}

		currentState.Instances[instance.Name] = state.InstanceState{
			Status:             "running",
			PID:                started.PID,
			ManagementIP:       defaultManagementHost,
			SSHPort:            sshPort,
			RuntimeDir:         runtimeDir,
			ProvisioningStatus: "ssh_ready",
			LastError:          "",
		}
		result.Instances = append(result.Instances, UpInstanceResult{
			Name:       instance.Name,
			Status:     "running",
			SSHAddress: address,
			SSHPort:    sshPort,
		})
	}

	if err := state.Save(paths.StateFile, currentState); err != nil {
		for i := len(startedInstances) - 1; i >= 0; i-- {
			_ = s.runtime.Stop(ctx, startedInstances[i], 5*time.Second)
		}
		return UpResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}

	sort.Slice(result.Instances, func(i, j int) bool {
		return result.Instances[i].Name < result.Instances[j].Name
	})
	return result, nil
}

func usedManagementPorts(currentState state.State) map[int]bool {
	used := make(map[int]bool, len(currentState.Instances))
	for _, instance := range currentState.Instances {
		if instance.SSHPort > 0 {
			used[instance.SSHPort] = true
		}
	}
	return used
}

func chooseManagementSSHPort(currentState state.State, instance config.Instance, used map[int]bool) (int, error) {
	if instance.SSHPort > 0 {
		if existing, ok := currentState.Instances[instance.Name]; ok && existing.SSHPort > 0 && existing.SSHPort != instance.SSHPort {
			return 0, fmt.Errorf("instance %q requested ssh_port %d but existing state uses %d", instance.Name, instance.SSHPort, existing.SSHPort)
		}
		if used[instance.SSHPort] {
			return 0, fmt.Errorf("requested ssh_port %d for instance %q is already in use", instance.SSHPort, instance.Name)
		}
		return instance.SSHPort, nil
	}
	if existing, ok := currentState.Instances[instance.Name]; ok && existing.SSHPort > 0 {
		return existing.SSHPort, nil
	}
	port := firstManagementSSHPort
	for used[port] {
		port++
	}
	return port, nil
}
