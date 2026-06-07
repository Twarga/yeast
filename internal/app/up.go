package app

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"yeast/internal/config"
	"yeast/internal/guest"
	"yeast/internal/images"
	"yeast/internal/project"
	"yeast/internal/provision"
	"yeast/internal/provision/cloudinit"
	provssh "yeast/internal/provision/ssh"
	rtm "yeast/internal/runtime"
	"yeast/internal/state"
)

const (
	defaultReadinessTimeout = 2 * time.Minute
	defaultBootstrapTimeout = 5 * time.Minute
	defaultManagementHost   = "127.0.0.1"
	defaultManagementNIC    = "yeastmgmt0"
	defaultLabInterfaceName = "yeastlab0"
	firstManagementSSHPort  = 2222
	managementStartAttempts = 3
)

type UpOptions struct {
	ProjectRoot      string
	ReadinessTimeout time.Duration
	Events           EventSink
}

type UpInstanceResult struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	SSHAddress string `json:"ssh_address,omitempty"`
	SSHPort    int    `json:"ssh_port,omitempty"`
	User       string `json:"user,omitempty"`
}

type UpResult struct {
	ProjectID string             `json:"project_id"`
	Instances []UpInstanceResult `json:"instances"`
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
	emitEvent(options.Events, "up", EventProjectLoaded, EventOptions{
		ProjectID: metadata.ID,
		Message:   "Project metadata loaded",
	})

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
	emitEvent(options.Events, "up", EventConfigValidated, EventOptions{
		ProjectID: metadata.ID,
		Message:   "Config loaded and validated",
		Data: map[string]any{
			"instances": len(cfg.Instances),
		},
	})

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
				User:       existing.User,
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
		emitEvent(options.Events, "up", EventImageReady, EventOptions{
			ProjectID: metadata.ID,
			Instance:  instance.Name,
			Message:   "Image is available",
			Data: map[string]any{
				"image": image.Name,
				"path":  cachePaths.ImageFile,
			},
		})

		runtimeDir, err := paths.InstanceDir(instance.Name)
		if err != nil {
			return UpResult{}, WrapError(ErrorCodeInvalidArgument, err.Error(), err)
		}
		portAvailable := s.managementPortAvailable
		if portAvailable == nil {
			portAvailable = managementPortAvailable
		}
		sshPort, err := s.chooseManagementSSHPort(currentState, instance, allocatedPorts, portAvailable)
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
		labNetworkPlan, err := buildLabNetworkPlan(cfg, instance, metadata.ID)
		if err != nil {
			return UpResult{}, WrapError(ErrorCodeInvalidArgument, fmt.Sprintf("build lab network plan for %s: %v", instance.Name, err), err)
		}
		var networkConfig string
		if labNetworkPlan != nil {
			networkConfig, err = s.renderNetworkConfig(cloudinit.NetworkConfigInput{
				ManagementInterfaceName: defaultManagementNIC,
				ManagementMACAddress:    deriveManagementMACAddress(metadata.ID, instance.Name),
				LabInterfaceName:        labNetworkPlan.InterfaceName,
				LabMACAddress:           labNetworkPlan.MACAddress,
				LabIPv4:                 labNetworkPlan.IPv4,
				LabCIDR:                 labNetworkPlan.CIDR,
			})
			if err != nil {
				return UpResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
			}
		}
		seedResult, err := s.createSeedISO(ctx, cloudinit.SeedInput{
			InstanceName:  instance.Name,
			RuntimeDir:    runtimeDir,
			UserData:      userData,
			MetaData:      metaData,
			NetworkConfig: networkConfig,
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
			Networks: rtm.NetworkPlan{
				Management: rtm.ManagementNetworkPlan{
					SSHHost:       defaultManagementHost,
					SSHPort:       sshPort,
					InterfaceName: defaultManagementNIC,
					MACAddress:    deriveManagementMACAddress(metadata.ID, instance.Name),
				},
				Lab: labNetworkPlan,
			},
		}

		provisionPlan, err := resolveProvisionPlan(absoluteRoot, provision.BuildPlan(instance, cfg.Provision))
		if err != nil {
			return UpResult{}, WrapError(ErrorCodeInvalidArgument, fmt.Sprintf("resolve provision plan for %s: %v", instance.Name, err), err)
		}
		if err := validateProvisionSudoPolicy(instance, provisionPlan); err != nil {
			return UpResult{}, WrapError(ErrorCodeInvalidArgument, err.Error(), err)
		}

		if _, err := s.runtime.PrepareDisk(ctx, plan); err != nil {
			return UpResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
		}
		emitEvent(options.Events, "up", EventDiskReady, EventOptions{
			ProjectID: metadata.ID,
			Instance:  instance.Name,
			Message:   "Disk is ready",
			Data: map[string]any{
				"disk_path": plan.Disk.DiskPath,
			},
		})

		emitEvent(options.Events, "up", EventVMStarting, EventOptions{
			ProjectID: metadata.ID,
			Instance:  instance.Name,
			Message:   "Starting VM",
		})
		started, err := s.startWithManagementPortRetry(ctx, plan, sshPort, instance.Name)
		if err != nil {
			return UpResult{}, err
		}
		startedInstances = append(startedInstances, started)

		address, err := s.sshAddress(defaultManagementHost, sshPort)
		if err != nil {
			_ = s.runtime.Stop(ctx, started, 5*time.Second)
			return UpResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
		}

		previousState, hadPreviousState := currentState.Instances[instance.Name]
		instanceState := state.InstanceState{
			Status:             "running",
			PID:                started.PID,
			ManagementIP:       defaultManagementHost,
			SSHPort:            sshPort,
			User:               instance.User,
			RuntimeDir:         runtimeDir,
			ProvisionLogPath:   filepath.Join(runtimeDir, "provision.log"),
			ProvisioningStatus: state.ProvisioningStatusNotStarted,
			LastError:          "",
		}
		if labNetworkPlan != nil {
			instanceState.LabIP = labNetworkPlan.IPv4.String()
		}
		if hadPreviousState && len(previousState.Snapshots) > 0 {
			instanceState.Snapshots = previousState.Snapshots
		}
		currentState.Instances[instance.Name] = instanceState
		if err := state.Save(paths.StateFile, currentState); err != nil {
			_ = s.runtime.Destroy(ctx, started)
			return UpResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
		}

		if err := s.waitForTCP(ctx, guest.ReadinessOptions{
			Address: address,
			Timeout: readinessTimeout,
		}); err != nil {
			message := fmt.Sprintf("wait for ssh readiness for %s at %s: %v", instance.Name, address, err)
			if destroyErr := s.runtime.Destroy(ctx, started); destroyErr != nil {
				instanceState.LastError = fmt.Sprintf("%s; cleanup failed: %v", message, destroyErr)
			} else {
				instanceState.Status = "stopped"
				instanceState.PID = 0
				instanceState.ManagementIP = ""
				instanceState.SSHPort = 0
				instanceState.ProvisioningStatus = state.ProvisioningStatusFailed
				instanceState.LastError = message
			}
			currentState.Instances[instance.Name] = instanceState
			if saveErr := state.Save(paths.StateFile, currentState); saveErr != nil {
				return UpResult{}, WrapError(ErrorCodeInternal, saveErr.Error(), saveErr)
			}
			return UpResult{}, WrapError(ErrorCodePrecondition, message, err)
		}
		emitEvent(options.Events, "up", EventSSHReady, EventOptions{
			ProjectID: metadata.ID,
			Instance:  instance.Name,
			Message:   "SSH is ready",
			Data: map[string]any{
				"ssh_address": address,
				"ssh_port":    sshPort,
			},
		})

		provisionLogPath := instanceState.ProvisionLogPath
		if provisionPlan.Empty() {
			instanceState.ProvisioningStatus = state.ProvisioningStatusReady
		} else {
			instanceState.ProvisioningStatus = state.ProvisioningStatusRunning
			emitEvent(options.Events, "up", EventProvisionStarted, EventOptions{
				ProjectID: metadata.ID,
				Instance:  instance.Name,
				Message:   "Provisioning started",
			})
		}
		currentState.Instances[instance.Name] = instanceState
		if err := state.Save(paths.StateFile, currentState); err != nil {
			return UpResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
		}

		provisionResult, err := s.runProvisionPlan(ctx, instance, sshPort, provisionLogPath, provisionPlan)
		instanceState.ProvisioningStatus = provisionResult.Status
		instanceState.LastError = ""
		if err != nil {
			instanceState.ProvisioningStatus = state.ProvisioningStatusFailed
			instanceState.LastError = err.Error()
			currentState.Instances[instance.Name] = instanceState
			if saveErr := state.Save(paths.StateFile, currentState); saveErr != nil {
				return UpResult{}, WrapError(ErrorCodeInternal, saveErr.Error(), saveErr)
			}
			appErr := NormalizeError(err)
			if appErr.Code == ErrorCodeInternal {
				return UpResult{}, appErr
			}
			return UpResult{}, WrapError(
				ErrorCodeProvisioning,
				fmt.Sprintf("provision instance %s: %v (log: %s)", instance.Name, err, provisionLogPath),
				err,
			)
		}
		emitEvent(options.Events, "up", EventProvisionFinished, EventOptions{
			ProjectID: metadata.ID,
			Instance:  instance.Name,
			Message:   "Provisioning finished",
			Data: map[string]any{
				"status": string(instanceState.ProvisioningStatus),
			},
		})
		currentState.Instances[instance.Name] = instanceState

		result.Instances = append(result.Instances, UpInstanceResult{
			Name:       instance.Name,
			Status:     "running",
			SSHAddress: address,
			SSHPort:    sshPort,
			User:       instance.User,
		})
		emitEvent(options.Events, "up", EventInstanceReady, EventOptions{
			ProjectID: metadata.ID,
			Instance:  instance.Name,
			Message:   "Instance is ready",
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
	emitEvent(options.Events, "up", EventWorkflowCompleted, EventOptions{
		ProjectID: metadata.ID,
		Message:   "Workflow completed",
	})
	return result, nil
}

func buildLabNetworkPlan(cfg *config.Config, instance config.Instance, projectID string) (*rtm.LabNetworkPlan, error) {
	if len(instance.Networks) == 0 {
		return nil, nil
	}

	attachment := instance.Networks[0]
	if strings.TrimSpace(attachment.Name) == "" {
		return nil, fmt.Errorf("network name is required")
	}

	var selected *config.Network
	for i := range cfg.Networks {
		if cfg.Networks[i].Name == attachment.Name {
			selected = &cfg.Networks[i]
			break
		}
	}
	if selected == nil {
		return nil, fmt.Errorf("unknown network %q", attachment.Name)
	}

	cidr, err := netip.ParsePrefix(strings.TrimSpace(selected.CIDR))
	if err != nil {
		return nil, fmt.Errorf("parse cidr %q: %w", selected.CIDR, err)
	}
	ipv4, err := netip.ParseAddr(strings.TrimSpace(attachment.IPv4))
	if err != nil {
		return nil, fmt.Errorf("parse ipv4 %q: %w", attachment.IPv4, err)
	}
	if !cidr.Contains(ipv4) {
		return nil, fmt.Errorf("ipv4 %s is outside cidr %s", ipv4, cidr)
	}

	return &rtm.LabNetworkPlan{
		Name:          selected.Name,
		CIDR:          cidr,
		IPv4:          ipv4,
		InterfaceName: defaultLabInterfaceName,
		MACAddress:    deriveLabMACAddress(projectID, instance.Name, selected.Name),
	}, nil
}

func deriveLabMACAddress(projectID, instanceName, networkName string) string {
	return deriveMACAddress(projectID, instanceName, networkName)
}

func deriveManagementMACAddress(projectID, instanceName string) string {
	return deriveMACAddress(projectID, instanceName, "management")
}

func deriveMACAddress(projectID, instanceName, networkName string) string {
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(projectID))
	_, _ = hash.Write([]byte("|"))
	_, _ = hash.Write([]byte(instanceName))
	_, _ = hash.Write([]byte("|"))
	_, _ = hash.Write([]byte(networkName))
	sum := hash.Sum32()

	return fmt.Sprintf(
		"52:54:%02x:%02x:%02x:%02x",
		byte(sum>>24),
		byte(sum>>16),
		byte(sum>>8),
		byte(sum),
	)
}

func (s *Service) startWithManagementPortRetry(ctx context.Context, plan rtm.MachinePlan, sshPort int, instanceName string) (rtm.RuntimeInstance, error) {
	sleep := s.sleep
	if sleep == nil {
		sleep = time.Sleep
	}

	cleanedOrphans := false
	for attempt := 1; attempt <= managementStartAttempts; attempt++ {
		started, err := s.runtime.Start(ctx, plan)
		if err != nil {
			return rtm.RuntimeInstance{}, WrapError(ErrorCodeInternal, err.Error(), err)
		}

		conflict, conflictErr := s.waitForManagementPortConflict(plan.LogPath, sshPort, time.Second)
		if conflictErr != nil {
			_ = s.runtime.Stop(ctx, started, 5*time.Second)
			return rtm.RuntimeInstance{}, WrapError(ErrorCodeInternal, conflictErr.Error(), conflictErr)
		}
		if !conflict {
			return started, nil
		}

		_ = s.runtime.Stop(ctx, started, 5*time.Second)
		if !cleanedOrphans {
			cleaned, cleanErr := s.cleanOrphanedQEMU(ctx, []rtm.CleanupTarget{{
				Name:       instanceName,
				RuntimeDir: plan.RuntimeDir,
				SSHHost:    plan.Networks.Management.SSHHost,
				SSHPort:    sshPort,
			}}, 5*time.Second)
			if cleanErr != nil {
				return rtm.RuntimeInstance{}, WrapError(ErrorCodeInternal, cleanErr.Error(), cleanErr)
			}
			if len(cleaned) > 0 {
				cleanedOrphans = true
				_ = s.waitForManagementPortRelease(ctx, sshPort, 5*time.Second)
				sleep(500 * time.Millisecond)
				continue
			}
		}
		if attempt == managementStartAttempts {
			return rtm.RuntimeInstance{}, WrapError(
				ErrorCodeInvalidArgument,
				fmt.Sprintf("requested ssh_port %d for instance %q is already bound on the host", sshPort, instanceName),
				nil,
			)
		}
		_ = s.waitForManagementPortRelease(ctx, sshPort, 2*time.Second)
		sleep(500 * time.Millisecond)
	}

	return rtm.RuntimeInstance{}, WrapError(
		ErrorCodeInvalidArgument,
		fmt.Sprintf("requested ssh_port %d for instance %q is already bound on the host", sshPort, instanceName),
		nil,
	)
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

func (s *Service) chooseManagementSSHPort(currentState state.State, instance config.Instance, used map[int]bool, portAvailable func(int) bool) (int, error) {
	if portAvailable == nil {
		portAvailable = managementPortAvailable
	}
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
	for used[port] || !portAvailable(port) {
		port++
	}
	return port, nil
}

func managementPortAvailable(port int) bool {
	if port <= 0 {
		return false
	}
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", defaultManagementHost, port))
	if err != nil {
		return false
	}
	_ = listener.Close()
	return true
}

func (s *Service) waitForManagementPortConflict(logPath string, port int, timeout time.Duration) (bool, error) {
	if logPath == "" || port <= 0 {
		return false, nil
	}
	sleep := s.sleep
	if sleep == nil {
		sleep = time.Sleep
	}
	deadline := time.Now().Add(timeout)
	conflictNeedle := fmt.Sprintf("tcp:%s:%d-:22", defaultManagementHost, port)

	for {
		content, err := os.ReadFile(logPath)
		if err == nil {
			text := string(content)
			if strings.Contains(text, "Could not set up host forwarding rule") && strings.Contains(text, conflictNeedle) {
				return true, nil
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return false, fmt.Errorf("read runtime log %s: %w", logPath, err)
		}
		if time.Now().After(deadline) {
			return false, nil
		}
		sleep(100 * time.Millisecond)
	}
}

func resolveProvisionPlan(projectRoot string, plan provision.Plan) (provision.Plan, error) {
	if plan.Empty() {
		return plan, nil
	}

	resolved := plan
	resolved.Files = make([]provision.FileStep, 0, len(plan.Files))
	for _, step := range plan.Files {
		source := step.Source
		if !filepath.IsAbs(source) {
			source = filepath.Join(projectRoot, source)
		}
		source = filepath.Clean(source)
		if _, err := os.Stat(source); err != nil {
			return provision.Plan{}, fmt.Errorf("file source %q: %w", step.Source, err)
		}
		resolved.Files = append(resolved.Files, provision.FileStep{
			Source:      source,
			Destination: step.Destination,
			Permissions: step.Permissions,
			Origin:      step.Origin,
		})
	}

	return resolved, nil
}

func (s *Service) runProvisionPlan(ctx context.Context, instance config.Instance, sshPort int, logPath string, plan provision.Plan) (provision.Result, error) {
	result := provision.NewResult(logPath)

	transport := s.provisionTransport
	if transport == nil {
		transport = provssh.NewLocalTransport()
	}

	user := instance.User
	host := defaultManagementHost

	bootstrapWaitResult, err := s.waitForCloudInitBootstrap(ctx, transport, provssh.RunRequest{
		User:    user,
		Host:    host,
		Port:    sshPort,
		Command: "cloud-init status --wait",
		Timeout: defaultBootstrapTimeout,
	})
	if bootstrapWaitResult.ExitCode != 0 || err != nil {
		now := time.Now().UTC()
		result.Steps = append(result.Steps, provision.StepResult{
			Kind:        provision.StepKindShell,
			Description: "cloud-init status --wait",
			ExitCode:    bootstrapWaitResult.ExitCode,
			Stdout:      bootstrapWaitResult.Stdout,
			Stderr:      bootstrapWaitResult.Stderr,
			StartedAt:   now.Add(-bootstrapWaitResult.Duration),
			FinishedAt:  now,
			Err:         errorString(err),
		})
		result.Status = state.ProvisioningStatusFailed
		writeErr := writeProvisionLog(logPath, result)
		if writeErr != nil {
			return result, WrapError(ErrorCodeInternal, writeErr.Error(), writeErr)
		}
		return result, WrapError(ErrorCodePrecondition, fmt.Sprintf("wait for cloud-init bootstrap: %v", err), err)
	}

	if plan.Empty() {
		result.Status = state.ProvisioningStatusReady
		if err := writeProvisionLog(logPath, result); err != nil {
			return result, err
		}
		return result, nil
	}

	appendPackageResult := func(pkgResult provssh.PackageResult, err error) {
		if len(pkgResult.Packages) == 0 && pkgResult.Command == "" && err == nil {
			return
		}
		now := time.Now().UTC()
		result.Steps = append(result.Steps, provision.StepResult{
			Kind:        provision.StepKindPackage,
			Description: strings.TrimSpace(pkgResult.Command),
			ExitCode:    pkgResult.Run.ExitCode,
			Stdout:      pkgResult.Run.Stdout,
			Stderr:      pkgResult.Run.Stderr,
			StartedAt:   now.Add(-pkgResult.Run.Duration),
			FinishedAt:  now,
			Err:         errorString(err),
		})
	}
	appendFileResult := func(fileResult provssh.FileResult, err error) {
		for _, step := range fileResult.Files {
			exitCode := 0
			stdout := step.Mkdir.Stdout
			stderr := step.Mkdir.Stderr
			startedAt := time.Now().UTC().Add(-step.Mkdir.Duration)
			finishedAt := time.Now().UTC()
			if step.Chmod.ExitCode != 0 || step.Chmod.Stdout != "" || step.Chmod.Stderr != "" || step.Chmod.Duration != 0 {
				exitCode = step.Chmod.ExitCode
				stdout = joinOutput(step.Mkdir.Stdout, step.Chmod.Stdout)
				stderr = joinOutput(step.Mkdir.Stderr, step.Chmod.Stderr)
				startedAt = time.Now().UTC().Add(-(step.Mkdir.Duration + step.Chmod.Duration))
			}
			result.Steps = append(result.Steps, provision.StepResult{
				Kind:        provision.StepKindFile,
				Description: fmt.Sprintf("%s -> %s", step.Source, step.Destination),
				ExitCode:    exitCode,
				Stdout:      stdout,
				Stderr:      stderr,
				StartedAt:   startedAt,
				FinishedAt:  finishedAt,
				Err:         errorString(err),
			})
		}
	}
	appendShellResult := func(shellResult provssh.ShellResult, err error) {
		for _, step := range shellResult.Steps {
			now := time.Now().UTC()
			result.Steps = append(result.Steps, provision.StepResult{
				Kind:        provision.StepKindShell,
				Description: step.Command,
				ExitCode:    step.Run.ExitCode,
				Stdout:      step.Run.Stdout,
				Stderr:      step.Run.Stderr,
				StartedAt:   now.Add(-step.Run.Duration),
				FinishedAt:  now,
				Err:         errorString(err),
			})
		}
	}

	packageProvisioner := provssh.NewPackageProvisioner(transport)
	packageResult, err := packageProvisioner.Install(ctx, provssh.PackageRequest{
		User:     user,
		Host:     host,
		Port:     sshPort,
		Packages: plan.Packages,
	})
	appendPackageResult(packageResult, err)
	if err != nil {
		result.Status = state.ProvisioningStatusFailed
		writeErr := writeProvisionLog(logPath, result)
		if writeErr != nil {
			return result, WrapError(ErrorCodeInternal, writeErr.Error(), writeErr)
		}
		return result, WrapError(ErrorCodePrecondition, fmt.Sprintf("package provisioning failed: %v", err), err)
	}

	fileProvisioner := provssh.NewFileProvisioner(transport)
	fileResult, err := fileProvisioner.Upload(ctx, provssh.FileRequest{
		User:  user,
		Host:  host,
		Port:  sshPort,
		Files: plan.Files,
	})
	appendFileResult(fileResult, err)
	if err != nil {
		result.Status = state.ProvisioningStatusFailed
		writeErr := writeProvisionLog(logPath, result)
		if writeErr != nil {
			return result, WrapError(ErrorCodeInternal, writeErr.Error(), writeErr)
		}
		return result, WrapError(ErrorCodePrecondition, fmt.Sprintf("file provisioning failed: %v", err), err)
	}

	shellProvisioner := provssh.NewShellProvisioner(transport)
	shellResult, err := shellProvisioner.Run(ctx, provssh.ShellRequest{
		User:     user,
		Host:     host,
		Port:     sshPort,
		Commands: plan.Shell,
	})
	appendShellResult(shellResult, err)
	if err != nil {
		result.Status = state.ProvisioningStatusFailed
		writeErr := writeProvisionLog(logPath, result)
		if writeErr != nil {
			return result, WrapError(ErrorCodeInternal, writeErr.Error(), writeErr)
		}
		return result, WrapError(ErrorCodePrecondition, fmt.Sprintf("shell provisioning failed: %v", err), err)
	}

	result.Status = state.ProvisioningStatusReady
	if err := writeProvisionLog(logPath, result); err != nil {
		return result, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	return result, nil
}

func (s *Service) waitForCloudInitBootstrap(ctx context.Context, transport provssh.Transport, request provssh.RunRequest) (provssh.RunResult, error) {
	sleep := s.sleep
	if sleep == nil {
		sleep = time.Sleep
	}

	deadline := time.Now().Add(request.Timeout)
	for {
		result, err := transport.Run(ctx, request)
		if err == nil && result.ExitCode == 0 {
			return result, nil
		}
		if ctx.Err() != nil || time.Now().After(deadline) {
			return result, err
		}
		sleep(2 * time.Second)
	}
}

func writeProvisionLog(logPath string, result provision.Result) error {
	if logPath == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return fmt.Errorf("create provision log directory for %s: %w", logPath, err)
	}

	var builder strings.Builder
	builder.WriteString("status: ")
	builder.WriteString(string(result.Status))
	builder.WriteString("\n")
	if len(result.Steps) == 0 {
		builder.WriteString("steps: 0\n")
	} else {
		builder.WriteString("steps:\n")
		for _, step := range result.Steps {
			builder.WriteString("- [")
			builder.WriteString(string(step.Kind))
			builder.WriteString("] ")
			builder.WriteString(step.Description)
			builder.WriteString(" exit=")
			builder.WriteString(fmt.Sprintf("%d", step.ExitCode))
			if step.Err != "" {
				builder.WriteString(" error=")
				builder.WriteString(step.Err)
			}
			builder.WriteString("\n")
			if trimmed := strings.TrimSpace(step.Stdout); trimmed != "" {
				builder.WriteString("  stdout:\n")
				for _, line := range strings.Split(trimmed, "\n") {
					builder.WriteString("    ")
					builder.WriteString(line)
					builder.WriteString("\n")
				}
			}
			if trimmed := strings.TrimSpace(step.Stderr); trimmed != "" {
				builder.WriteString("  stderr:\n")
				for _, line := range strings.Split(trimmed, "\n") {
					builder.WriteString("    ")
					builder.WriteString(line)
					builder.WriteString("\n")
				}
			}
		}
	}

	if err := os.WriteFile(logPath, []byte(builder.String()), 0644); err != nil {
		return fmt.Errorf("write provision log %s: %w", logPath, err)
	}
	return nil
}

func joinOutput(parts ...string) string {
	combined := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part) != "" {
			combined = append(combined, strings.TrimRight(part, "\n"))
		}
	}
	if len(combined) == 0 {
		return ""
	}
	return strings.Join(combined, "\n") + "\n"
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
