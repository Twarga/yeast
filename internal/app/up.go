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
	"sync"
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
	cloudInitStatusCommand  = `status="$(cloud-init status 2>&1)"; printf '%s\n' "$status"; case "$status" in *"status: done"*|*"status: disabled"*) exit 0;; *"status: error"*) exit 2;; *) exit 1;; esac`
	defaultManagementHost   = "127.0.0.1"
	defaultManagementNIC    = "yeastmgmt0"
	defaultLabInterfaceName = "yeastlab0"
	firstManagementSSHPort  = 2222
	managementStartAttempts = 3
)

func resolveManagementHost(cfg *config.Config) string {
	if cfg != nil && strings.TrimSpace(cfg.ManagementHost) != "" {
		return strings.TrimSpace(cfg.ManagementHost)
	}
	return defaultManagementHost
}

type UpOptions struct {
	ProjectRoot      string
	ReadinessTimeout time.Duration
	NoProvision      bool
	Reprovision      bool
	Sequential       bool
	Profile          bool
	Events           EventSink
}

type UpInstanceResult struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	SSHAddress string `json:"ssh_address,omitempty"`
	SSHPort    int    `json:"ssh_port,omitempty"`
}

type UpResult struct {
	ProjectID string             `json:"project_id"`
	Instances []UpInstanceResult `json:"instances"`
	Profile   *ProfileData       `json:"profile,omitempty"`
}

type ProfileData struct {
	Phases []ProfilePhase `json:"phases"`
	Total  time.Duration  `json:"total_ms"`
}

type ProfilePhase struct {
	Name     string        `json:"name"`
	Duration time.Duration `json:"duration_ms"`
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

	managementHost := resolveManagementHost(cfg)

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

	var profilePhases []ProfilePhase
	profileStart := time.Now()
	lastPhaseStart := profileStart
	finishProfilePhase := func(name string) {
		if options.Profile {
			now := time.Now()
			profilePhases = append(profilePhases, ProfilePhase{
				Name:     name,
				Duration: now.Sub(lastPhaseStart),
			})
			lastPhaseStart = now
		}
	}

	allocatedPorts := usedManagementPorts(currentState)
	startedInstances := make([]rtm.RuntimeInstance, 0)

	type bootPlan struct {
		instance         config.Instance
		plan             rtm.MachinePlan
		sshPort          int
		instanceState    state.InstanceState
		previousState    state.InstanceState
		hadPreviousState bool
		provisionPlan    provision.Plan
		runProvision     bool
		fingerprint      string
	}

	var plans []bootPlan

	for _, instance := range cfg.Instances {
		if existing, ok := currentState.Instances[instance.Name]; ok && existing.Status == "running" && existing.PID > 0 && existing.SSHPort > 0 {
			address, err := s.sshAddress(managementHost, existing.SSHPort)
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

			if !options.NoProvision {
				existingPlan, planErr := resolveProvisionPlan(absoluteRoot, provision.BuildPlan(instance, cfg.Provision))
				if planErr != nil {
					return UpResult{}, WrapError(ErrorCodeInvalidArgument, fmt.Sprintf("resolve provision plan for %s: %v", instance.Name, planErr), planErr)
				}
				if !existingPlan.Empty() {
					fp, _ := provision.Fingerprint(absoluteRoot, instance, cfg)
					prevFP := existing.ProvisionFingerprint
					fpMatch := prevFP != "" && fp == prevFP
					wasReady := existing.ProvisioningStatus == state.ProvisioningStatusReady
					needsReprovision := !wasReady || !fpMatch || options.Reprovision
					if needsReprovision {
						instState := existing
						instState.ProvisioningStatus = state.ProvisioningStatusRunning
						instState.ProvisionLogPath = filepath.Join(existing.RuntimeDir, "provision.log")
						currentState.Instances[instance.Name] = instState
						plans = append(plans, bootPlan{
							instance:         instance,
							sshPort:          existing.SSHPort,
							instanceState:    instState,
							previousState:    existing,
							hadPreviousState: true,
							provisionPlan:    existingPlan,
							runProvision:     true,
							fingerprint:      fp,
						})
					}
				}
			}

			continue
		}

		image, ok := images.Lookup(instance.Image)
		if !ok {
			matches := images.Search(instance.Image)
			if len(matches) == 1 {
				image, _ = images.Lookup(matches[0])
			} else if len(matches) > 1 {
				msg := fmt.Sprintf("multiple images match %q:", instance.Image)
				for _, m := range matches {
					msg += fmt.Sprintf("\n  - %s", m)
				}
				msg += "\n\nSpecify the full name in yeast.yaml."
				return UpResult{}, WrapError(ErrorCodeInvalidArgument, msg, nil)
			} else {
				msg := fmt.Sprintf("image %q not found", instance.Image)
				suggestions := images.SuggestSimilar(instance.Image, 3)
				if len(suggestions) > 0 {
					msg += "\n\nDid you mean?\n"
					for _, s := range suggestions {
						msg += fmt.Sprintf("  - %s\n", s)
					}
				}
				msg += "\nRun \"yeast pull --list\" for all available images."
				return UpResult{}, WrapError(ErrorCodeInvalidArgument, msg, nil)
			}
		}

		// Manual images — show instructions instead of downloading.
		if image.URL == "" && image.ManualInstructions != "" {
			msg := fmt.Sprintf("image %q requires manual download:\n\n%s", image.Name, image.ManualInstructions)
			return UpResult{}, WrapError(ErrorCodeInvalidArgument, msg, nil)
		}

		cachePaths, err := images.ResolveCachePaths(paths.ImageCache, image.Name)
		if err != nil {
			return UpResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
		}
		if _, err := os.Stat(cachePaths.ImageFile); err != nil {
			emitEvent(options.Events, "up", EventImagePulling, EventOptions{
				ProjectID: metadata.ID,
				Instance:  instance.Name,
				Message:   fmt.Sprintf("Pulling image %s", image.Name),
				Data: map[string]any{
					"image": image.Name,
				},
			})
			if pullErr := s.downloadImage(image, cachePaths.ImageFile, s.downloadOptions()); pullErr != nil {
				return UpResult{}, WrapError(ErrorCodeInternal,
					fmt.Sprintf("auto-pull image %s failed: %v", image.Name, pullErr), pullErr)
			}
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
			portAvailable = func(port int) bool {
				return managementPortAvailable(managementHost, port)
			}
		}
		sshPort, err := s.chooseManagementSSHPort(currentState, instance, allocatedPorts, portAvailable, managementHost)
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
		networkInput := cloudinit.NetworkConfigInput{
			ManagementInterfaceName: defaultManagementNIC,
			ManagementMACAddress:    deriveManagementMACAddress(metadata.ID, instance.Name),
		}
		if labNetworkPlan != nil {
			networkInput.LabInterfaceName = labNetworkPlan.InterfaceName
			networkInput.LabMACAddress = labNetworkPlan.MACAddress
			networkInput.LabIPv4 = labNetworkPlan.IPv4
			networkInput.LabCIDR = labNetworkPlan.CIDR
		}
		networkConfig, err := s.renderNetworkConfig(networkInput)
		if err != nil {
			return UpResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
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
					SSHHost:       managementHost,
					SSHPort:       sshPort,
					InterfaceName: defaultManagementNIC,
					MACAddress:    deriveManagementMACAddress(metadata.ID, instance.Name),
				},
				Lab: labNetworkPlan,
			},
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

		previousState, hadPreviousState := currentState.Instances[instance.Name]
		previousProvisioningStatus := state.ProvisioningStatusNotStarted
		if hadPreviousState {
			previousProvisioningStatus = previousState.ProvisioningStatus
		}
		instState := state.InstanceState{
			Status:             "running",
			ManagementIP:       managementHost,
			SSHPort:            sshPort,
			User:               instance.User,
			RuntimeDir:         runtimeDir,
			ProvisionLogPath:   filepath.Join(runtimeDir, "provision.log"),
			ProvisioningStatus: previousProvisioningStatus,
			LastError:          "",
		}
		if labNetworkPlan != nil {
			instState.LabIP = labNetworkPlan.IPv4.String()
		}
		if hadPreviousState && len(previousState.Snapshots) > 0 {
			instState.Snapshots = previousState.Snapshots
		}

		provisionPlan, err := resolveProvisionPlan(absoluteRoot, provision.BuildPlan(instance, cfg.Provision))
		if err != nil {
			return UpResult{}, WrapError(ErrorCodeInvalidArgument, fmt.Sprintf("resolve provision plan for %s: %v", instance.Name, err), err)
		}
		runProvisionSteps := !options.NoProvision && !provisionPlan.Empty()

		currentFingerprint, _ := provision.Fingerprint(absoluteRoot, instance, cfg)
		previousFingerprint := ""
		if hadPreviousState {
			previousFingerprint = previousState.ProvisionFingerprint
		}
		isWarmBoot := instState.ProvisioningStatus == state.ProvisioningStatusReady
		fingerprintMatch := previousFingerprint != "" && currentFingerprint == previousFingerprint
		if runProvisionSteps && isWarmBoot && fingerprintMatch && !options.Reprovision {
			runProvisionSteps = false
			instState.ProvisionFingerprint = currentFingerprint
			emitEvent(options.Events, "up", EventProvisionSkipped, EventOptions{
				ProjectID: metadata.ID,
				Instance:  instance.Name,
				Message:   "Provisioning skipped (unchanged)",
			})
		}
		if options.NoProvision || provisionPlan.Empty() {
			instState.ProvisioningStatus = state.ProvisioningStatusReady
		} else if runProvisionSteps {
			instState.ProvisioningStatus = state.ProvisioningStatusRunning
			emitEvent(options.Events, "up", EventProvisionStarted, EventOptions{
				ProjectID: metadata.ID,
				Instance:  instance.Name,
				Message:   "Provisioning started",
			})
		}
		currentState.Instances[instance.Name] = instState

		plans = append(plans, bootPlan{
			instance:         instance,
			plan:             plan,
			sshPort:          sshPort,
			instanceState:    instState,
			previousState:    previousState,
			hadPreviousState: hadPreviousState,
			provisionPlan:    provisionPlan,
			runProvision:     runProvisionSteps,
			fingerprint:      currentFingerprint,
		})
	}

	finishProfilePhase("disk_prepare")

	if err := state.Save(paths.StateFile, currentState); err != nil {
		return UpResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}

	type bootResult struct {
		name    string
		result  UpInstanceResult
		started rtm.RuntimeInstance
		err     error
	}

	bootVM := func(bp bootPlan) bootResult {
		emitEvent(options.Events, "up", EventVMStarting, EventOptions{
			ProjectID: metadata.ID,
			Instance:  bp.instance.Name,
			Message:   "Starting VM",
		})
		started, err := s.startWithManagementPortRetry(ctx, bp.plan, bp.sshPort, bp.instance.Name, managementHost)
		if err != nil {
			return bootResult{name: bp.instance.Name, err: err}
		}

		address, err := s.sshAddress(managementHost, bp.sshPort)
		if err != nil {
			_ = s.runtime.Stop(ctx, started, 5*time.Second)
			return bootResult{name: bp.instance.Name, err: err}
		}
		if err := s.waitForTCP(ctx, guest.ReadinessOptions{
			Address: address,
			Timeout: readinessTimeout,
		}); err != nil {
			_ = s.runtime.Stop(ctx, started, 5*time.Second)
			return bootResult{
				name: bp.instance.Name,
				err:  WrapError(ErrorCodePrecondition, fmt.Sprintf("wait for ssh readiness for %s: %v", bp.instance.Name, err), err),
			}
		}
		if err := s.ensureRuntimeStillRunning(ctx, started, bp.instance.Name); err != nil {
			_ = s.runtime.Stop(ctx, started, 5*time.Second)
			return bootResult{name: bp.instance.Name, err: err}
		}
		emitEvent(options.Events, "up", EventSSHReady, EventOptions{
			ProjectID: metadata.ID,
			Instance:  bp.instance.Name,
			Message:   "SSH is ready",
			Data: map[string]any{
				"ssh_address": address,
				"ssh_port":    bp.sshPort,
			},
		})

		return bootResult{
			name:    bp.instance.Name,
			started: started,
			result: UpInstanceResult{
				Name:       bp.instance.Name,
				Status:     "running",
				SSHAddress: address,
				SSHPort:    bp.sshPort,
			},
		}
	}

	var bootResults []bootResult
	if options.Sequential || len(plans) <= 1 {
		for _, bp := range plans {
			bootResults = append(bootResults, bootVM(bp))
		}
	} else {
		bootResults = make([]bootResult, len(plans))
		var wg sync.WaitGroup
		for i, bp := range plans {
			wg.Add(1)
			go func(idx int, p bootPlan) {
				defer wg.Done()
				bootResults[idx] = bootVM(p)
			}(i, bp)
		}
		wg.Wait()
	}

	for _, br := range bootResults {
		if br.err != nil {
			for i := len(startedInstances) - 1; i >= 0; i-- {
				_ = s.runtime.Stop(ctx, startedInstances[i], 5*time.Second)
			}
			if _, ok := br.err.(*AppError); !ok {
				return UpResult{}, WrapError(ErrorCodeInternal, br.err.Error(), br.err)
			}
			return UpResult{}, br.err
		}
		startedInstances = append(startedInstances, br.started)
		result.Instances = append(result.Instances, br.result)

		for _, bp := range plans {
			if bp.instance.Name == br.name {
				bp.instanceState.PID = br.started.PID
				currentState.Instances[bp.instance.Name] = bp.instanceState
				break
			}
		}
	}
	if err := state.Save(paths.StateFile, currentState); err != nil {
		for i := len(startedInstances) - 1; i >= 0; i-- {
			_ = s.runtime.Stop(ctx, startedInstances[i], 5*time.Second)
		}
		return UpResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}

	finishProfilePhase("vm_boot")

	type provisionResult struct {
		name        string
		status      state.ProvisioningStatus
		fingerprint string
		err         error
	}

	provisionVM := func(bp bootPlan) provisionResult {
		pResult, err := s.runProvisionPlan(ctx, bp.instance, bp.sshPort, bp.instanceState.ProvisionLogPath, bp.provisionPlan, bp.runProvision, managementHost)
		if err != nil {
			return provisionResult{name: bp.instance.Name, status: state.ProvisioningStatusFailed, err: err}
		}
		fp := bp.fingerprint
		if bp.runProvision {
			fp = bp.fingerprint
		}
		return provisionResult{name: bp.instance.Name, status: pResult.Status, fingerprint: fp}
	}

	var provisionResults []provisionResult
	if options.Sequential || len(plans) <= 1 {
		for _, bp := range plans {
			provisionResults = append(provisionResults, provisionVM(bp))
		}
	} else {
		provisionResults = make([]provisionResult, len(plans))
		var wg sync.WaitGroup
		for i, bp := range plans {
			wg.Add(1)
			go func(idx int, p bootPlan) {
				defer wg.Done()
				provisionResults[idx] = provisionVM(p)
			}(i, bp)
		}
		wg.Wait()
	}

	for _, pr := range provisionResults {
		instState := currentState.Instances[pr.name]
		instState.ProvisioningStatus = pr.status
		instState.LastError = ""
		if pr.err != nil {
			instState.ProvisioningStatus = state.ProvisioningStatusFailed
			instState.LastError = pr.err.Error()
			currentState.Instances[pr.name] = instState
			if saveErr := state.Save(paths.StateFile, currentState); saveErr != nil {
				return UpResult{}, WrapError(ErrorCodeInternal, saveErr.Error(), saveErr)
			}
			appErr := NormalizeError(pr.err)
			if appErr.Code == ErrorCodeInternal {
				return UpResult{}, appErr
			}
			return UpResult{}, WrapError(ErrorCodeProvisioning, fmt.Sprintf("provision instance %s: %v", pr.name, pr.err), pr.err)
		}
		if pr.fingerprint != "" {
			instState.ProvisionFingerprint = pr.fingerprint
		}
		emitEvent(options.Events, "up", EventProvisionFinished, EventOptions{
			ProjectID: metadata.ID,
			Instance:  pr.name,
			Message:   "Provisioning finished",
			Data: map[string]any{
				"status": string(instState.ProvisioningStatus),
			},
		})
		currentState.Instances[pr.name] = instState
	}

	finishProfilePhase("provision")

	for _, bp := range plans {
		emitEvent(options.Events, "up", EventInstanceReady, EventOptions{
			ProjectID: metadata.ID,
			Instance:  bp.instance.Name,
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

	if options.Profile {
		totalDuration := time.Since(profileStart)
		result.Profile = &ProfileData{
			Phases: profilePhases,
			Total:  totalDuration,
		}
	}

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

func (s *Service) startWithManagementPortRetry(ctx context.Context, plan rtm.MachinePlan, sshPort int, instanceName string, managementHost string) (rtm.RuntimeInstance, error) {
	sleep := s.sleep
	if sleep == nil {
		sleep = time.Sleep
	}

	for attempt := 1; attempt <= managementStartAttempts; attempt++ {
		started, err := s.runtime.Start(ctx, plan)
		if err != nil {
			return rtm.RuntimeInstance{}, WrapError(ErrorCodeInternal, err.Error(), err)
		}

		conflict, conflictErr := s.waitForManagementPortConflict(plan.LogPath, sshPort, time.Second, managementHost)
		if conflictErr != nil {
			_ = s.runtime.Stop(ctx, started, 5*time.Second)
			return rtm.RuntimeInstance{}, WrapError(ErrorCodeInternal, conflictErr.Error(), conflictErr)
		}
		if !conflict {
			return started, nil
		}

		_ = s.runtime.Stop(ctx, started, 5*time.Second)
		if attempt == managementStartAttempts {
			return rtm.RuntimeInstance{}, WrapError(
				ErrorCodeInvalidArgument,
				fmt.Sprintf("requested ssh_port %d for instance %q is already bound on the host", sshPort, instanceName),
				nil,
			)
		}
		_ = s.waitForManagementPortRelease(ctx, managementHost, sshPort, 2*time.Second)
		sleep(500 * time.Millisecond)
	}

	return rtm.RuntimeInstance{}, WrapError(
		ErrorCodeInvalidArgument,
		fmt.Sprintf("requested ssh_port %d for instance %q is already bound on the host", sshPort, instanceName),
		nil,
	)
}

func (s *Service) ensureRuntimeStillRunning(ctx context.Context, instance rtm.RuntimeInstance, instanceName string) error {
	info, err := s.runtime.Inspect(ctx, instance)
	if err != nil {
		return WrapError(ErrorCodeInternal, fmt.Sprintf("inspect runtime for %s: %v", instanceName, err), err)
	}
	if info.State != rtm.ProcessStateRunning {
		return WrapError(ErrorCodePrecondition, fmt.Sprintf("instance %q exited before ssh became ready", instanceName), nil)
	}
	return nil
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

func (s *Service) chooseManagementSSHPort(currentState state.State, instance config.Instance, used map[int]bool, portAvailable func(int) bool, managementHost string) (int, error) {
	if portAvailable == nil {
		portAvailable = func(port int) bool {
			return managementPortAvailable(managementHost, port)
		}
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

func managementPortAvailable(host string, port int) bool {
	if port <= 0 {
		return false
	}
	if strings.TrimSpace(host) == "" {
		host = defaultManagementHost
	}
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return false
	}
	_ = listener.Close()
	return true
}

func (s *Service) waitForManagementPortConflict(logPath string, port int, timeout time.Duration, managementHost string) (bool, error) {
	if logPath == "" || port <= 0 {
		return false, nil
	}
	sleep := s.sleep
	if sleep == nil {
		sleep = time.Sleep
	}
	deadline := time.Now().Add(timeout)
	conflictNeedle := fmt.Sprintf("tcp:%s:%d-:22", managementHost, port)

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

func (s *Service) runProvisionPlan(ctx context.Context, instance config.Instance, sshPort int, logPath string, plan provision.Plan, runSteps bool, managementHost string) (provision.Result, error) {
	result := provision.NewResult(logPath)

	transport := s.provisionTransport
	if transport == nil {
		lt := provssh.NewLocalTransport()
		defer lt.Close()
		transport = lt
	}

	user := instance.User
	host := managementHost

	bootstrapWaitResult, err := s.waitForCloudInitBootstrap(ctx, transport, provssh.RunRequest{
		User:    user,
		Host:    host,
		Port:    sshPort,
		Command: cloudInitStatusCommand,
		Timeout: defaultBootstrapTimeout,
	})
	if bootstrapWaitResult.ExitCode != 0 || err != nil {
		now := time.Now().UTC()
		result.Steps = append(result.Steps, provision.StepResult{
			Kind:        provision.StepKindShell,
			Description: cloudInitStatusCommand,
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

	if plan.Empty() || !runSteps {
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
	probeTimeout := 15 * time.Second
	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return provssh.RunResult{}, fmt.Errorf("cloud-init did not finish within %s", request.Timeout)
		}
		probe := request
		if probe.Timeout <= 0 || probe.Timeout > probeTimeout {
			probe.Timeout = probeTimeout
		}
		if probe.Timeout > remaining {
			probe.Timeout = remaining
		}
		result, err := transport.Run(ctx, probe)
		if err == nil && result.ExitCode == 0 {
			return result, nil
		}
		if result.ExitCode == 2 || cloudInitStatusFailed(result.Stdout) || cloudInitStatusFailed(result.Stderr) {
			return result, fmt.Errorf("cloud-init reported failure")
		}
		if ctx.Err() != nil {
			return result, err
		}
		sleep(2 * time.Second)
	}
}

func cloudInitStatusFailed(output string) bool {
	return strings.Contains(strings.ToLower(output), "status: error")
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
