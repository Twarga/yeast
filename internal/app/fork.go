package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Twarga/yeast/internal/project"
	rtm "github.com/Twarga/yeast/internal/runtime"
	"github.com/Twarga/yeast/internal/state"
)

const maxForksPerCall = 32

type ForkOptions struct {
	ProjectRoot string
	Target      string
	Count       int
	From        string // optional snapshot name to fork from
	Boot        bool   // start the forks immediately
	Events      EventSink
}

type ForkEntry struct {
	Name       string `json:"name"`
	RuntimeDir string `json:"runtime_dir"`
	SSHPort    int    `json:"ssh_port"`
	PID        int    `json:"pid,omitempty"`
	Status     string `json:"status"`
}

type ForkResult struct {
	ProjectID string      `json:"project_id"`
	Source    string      `json:"source"`
	Forks     []ForkEntry `json:"forks"`
}

// Fork creates N copy-on-write siblings of an instance. Disks are thin
// overlays on a frozen base: an existing snapshot (--from), or an implicit
// overlay snapshot of the stopped instance. Forks are registered in state
// with ForkOf set and can be managed like normal instances.
func (s *Service) Fork(ctx context.Context, options ForkOptions) (out ForkResult, err error) {
	started := time.Now()
	root := options.ProjectRoot
	if root == "" {
		root = "."
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return ForkResult{}, WrapError(ErrorCodeInternal, fmt.Sprintf("resolve project root: %v", err), err)
	}
	defer func() {
		s.auditCall(ctx, absoluteRoot, "fork", map[string]any{"instance": options.Target, "count": options.Count}, started, err)
	}()
	if strings.TrimSpace(options.Target) == "" {
		return ForkResult{}, WrapError(ErrorCodeInvalidArgument, "fork source instance is required", nil)
	}
	count := options.Count
	if count <= 0 {
		count = 1
	}
	if count > maxForksPerCall {
		return ForkResult{}, WrapError(ErrorCodeInvalidArgument, fmt.Sprintf("at most %d forks per call", maxForksPerCall), nil)
	}

	metadata, err := project.LoadMetadata(absoluteRoot)
	if err != nil {
		if errors.Is(err, project.ErrMetadataNotFound) {
			return ForkResult{}, WrapError(ErrorCodePrecondition, err.Error(), err)
		}
		return ForkResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	yeastHome, err := s.resolveYeastHome()
	if err != nil {
		return ForkResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	paths, err := project.NewPaths(yeastHome, metadata)
	if err != nil {
		return ForkResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	lock, err := state.Acquire(paths.StateLock, state.DefaultLockOptions())
	if err != nil {
		return ForkResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	defer func() { _ = lock.Release() }()

	currentState, err := state.Load(paths.StateFile, metadata.ID)
	if err != nil {
		return ForkResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	parent, ok := currentState.Instances[options.Target]
	if !ok {
		return ForkResult{}, WrapError(ErrorCodeNotFound, fmt.Sprintf("instance %q not found", options.Target), nil)
	}

	// The boot plan recorded at `yeast up` carries the exact topology.
	parentPlan, err := loadBootPlan(parent.RuntimeDir)
	if err != nil {
		return ForkResult{}, WrapError(ErrorCodePrecondition, fmt.Sprintf("instance %q has no recorded machine plan — run 'yeast up %s' once before forking (%v)", options.Target, options.Target, err), nil)
	}

	// Resolve the frozen base to fork from.
	base, err := s.forkSourceBase(ctx, options, parent, paths, metadata, &currentState)
	if err != nil {
		return ForkResult{}, err
	}

	used := usedHostBindings(currentState)
	// usedHostBindings only binds running instances' management IPs — also
	// block every recorded SSH port on the loopback host so stopped
	// instances keep their ports.
	for instName, inst := range currentState.Instances {
		if inst.SSHPort > 0 && inst.ManagementIP == "" {
			used[hostPortKey("127.0.0.1", inst.SSHPort)] = instName + " ssh"
		}
	}
	created := []string{}
	rollback := func() {
		for _, name := range created {
			rtDir, pid := "", 0
			if inst, exists := currentState.Instances[name]; exists {
				rtDir, pid = inst.RuntimeDir, inst.PID
			} else if d, err := paths.InstanceDir(name); err == nil {
				rtDir = d
			}
			_ = s.runtime.Destroy(ctx, rtm.RuntimeInstance{Name: name, RuntimeDir: rtDir, PID: pid})
			delete(currentState.Instances, name)
		}
		_ = state.Save(paths.StateFile, currentState)
	}

	result := ForkResult{ProjectID: metadata.ID, Source: options.Target}
	for i := 0; i < count; i++ {
		name := nextForkName(currentState, options.Target)
		rtDir, err := paths.InstanceDir(name)
		if err != nil {
			rollback()
			return ForkResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
		}

		plan := parentPlan
		plan.Name = name
		plan.RuntimeDir = rtDir
		plan.LogPath = filepath.Join(rtDir, "vm.log")
		plan.Disk.DiskPath = filepath.Join(rtDir, "disk.qcow2")
		plan.Disk.BaseImagePath = base
		plan.SeedImagePath = filepath.Join(rtDir, "seed.iso")
		plan.Networks.Management.SSHPort = allocFreeHostPort(used, plan.Networks.Management.SSHHost, firstManagementSSHPort)
		plan.Networks.Management.MACAddress = deriveManagementMACAddress(metadata.ID, name)
		used[hostPortKey(plan.Networks.Management.SSHHost, plan.Networks.Management.SSHPort)] = name + " ssh"
		plan.Networks.Management.PortForwards = reallocPortForwards(plan.Networks.Management.PortForwards, used, name)
		if plan.Display != nil {
			plan.Display.Port = allocFreeHostPort(used, plan.Display.Host, 5901)
			used[hostPortKey(plan.Display.Host, plan.Display.Port)] = name + " display"
		}
		plan.Networks.Lab = nil // forks are isolated; a shared lab needs explicit config

		if err := os.MkdirAll(rtDir, 0755); err != nil {
			rollback()
			return ForkResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
		}
		// Reuse the parent's seed ISO (cloud-init identity is per-disk anyway).
		if parentPlan.SeedImagePath != "" {
			if err := copySmallFile(parentPlan.SeedImagePath, plan.SeedImagePath); err != nil && !os.IsNotExist(err) {
				rollback()
				return ForkResult{}, WrapError(ErrorCodeInternal, fmt.Sprintf("copy seed iso: %v", err), err)
			}
		}
		// Copy UEFI NVRAM and egress allowlist when present so forked VMs boot identically.
		for _, extraFile := range []string{"ovmf_vars.fd", "egress.allowlist"} {
			src := filepath.Join(parent.RuntimeDir, extraFile)
			if _, statErr := os.Stat(src); statErr == nil {
				_ = copySmallFile(src, filepath.Join(rtDir, extraFile))
			}
		}
		created = append(created, name) // track before PrepareDisk for rollback
		if _, err := s.runtime.PrepareDisk(ctx, plan); err != nil {
			rollback()
			return ForkResult{}, WrapError(ErrorCodeInternal, fmt.Sprintf("create fork overlay: %v", err), err)
		}
		_ = writeBootPlan(rtDir, plan)

		entry := state.InstanceState{
			Status:             "stopped",
			RuntimeDir:         rtDir,
			SSHPort:            plan.Networks.Management.SSHPort,
			User:               parent.User,
			ForkOf:             options.Target,
			ProvisioningStatus: state.ProvisioningStatusReady,
			ServicePorts:       statePortForwardsFromPlan(plan.Networks.Management.PortForwards),
		}
		if plan.Display != nil {
			entry.Display = &state.DisplayState{
				Type: plan.Display.Type, Host: plan.Display.Host, Port: plan.Display.Port,
				URL:      fmt.Sprintf("vnc://%s:%d", plan.Display.Host, plan.Display.Port),
				GPU:      plan.Display.GPU,
				Firmware: plan.Firmware,
			}
		}
		currentState.Instances[name] = entry

		fe := ForkEntry{Name: name, RuntimeDir: rtDir, SSHPort: plan.Networks.Management.SSHPort, Status: "stopped"}
		if options.Boot {
			started, err := s.runtime.Start(ctx, plan)
			if err != nil {
				rollback()
				return ForkResult{}, WrapError(ErrorCodeInternal, fmt.Sprintf("boot fork %s: %v", name, err), err)
			}
			entry.Status = "running"
			entry.PID = started.PID
			currentState.Instances[name] = entry
			fe.Status = "running"
			fe.PID = started.PID
		}
		result.Forks = append(result.Forks, fe)
	}
	if err := state.Save(paths.StateFile, currentState); err != nil {
		return ForkResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	emitEvent(options.Events, "fork", EventForkCreated, EventOptions{
		ProjectID: metadata.ID,
		Instance:  options.Target,
		Message:   fmt.Sprintf("Created %d fork(s)", len(result.Forks)),
	})
	return result, nil
}

// forkSourceBase resolves the frozen qcow2 to back the forks. --from uses an
// existing snapshot (overlay/live base, or a legacy copy file); otherwise an
// implicit overlay snapshot is taken of the stopped instance.
func (s *Service) forkSourceBase(ctx context.Context, options ForkOptions, parent state.InstanceState, paths project.Paths, metadata project.Metadata, currentState *state.State) (string, error) {
	if options.From != "" {
		snap, ok := parent.Snapshots[options.From]
		if !ok {
			return "", WrapError(ErrorCodeNotFound, fmt.Sprintf("snapshot %q not found for instance %q", options.From, options.Target), nil)
		}
		if snap.Kind == "overlay" || snap.Kind == "live" {
			base := filepath.Join(snap.Dir, "base.qcow2")
			if _, err := os.Stat(base); err == nil {
				return base, nil
			}
			if matches, _ := filepath.Glob(filepath.Join(snap.Dir, "base-*.qcow2")); len(matches) == 1 {
				return matches[0], nil
			}
			return "", WrapError(ErrorCodeNotFound, fmt.Sprintf("snapshot %q frozen disk missing", options.From), nil)
		}
		return snap.DiskPath, nil
	}
	if parent.Status != "stopped" {
		return "", WrapError(ErrorCodePrecondition, fmt.Sprintf("instance %q is running — stop it or pass --from <snapshot>", options.Target), nil)
	}
	// Implicit seed snapshot so the parent keeps working on a fresh overlay.
	seedName := fmt.Sprintf("forkseed-%d", time.Now().UTC().Unix())
	snapDir, err := paths.SnapshotDir(options.Target)
	if err != nil {
		return "", WrapError(ErrorCodeInternal, err.Error(), err)
	}
	snapDir = filepath.Join(snapDir, seedName)
	base := filepath.Join(snapDir, "base.qcow2")
	if err := s.runtime.CreateSnapshot(ctx, rtm.SnapshotPlan{
		InstanceDiskPath: activeDiskPath(parent),
		SnapshotPath:     base,
		Kind:             "overlay",
		RuntimeDir:       parent.RuntimeDir,
	}); err != nil {
		return "", WrapError(ErrorCodeInternal, err.Error(), err)
	}
	parent.ActiveDisk = ""
	if parent.Snapshots == nil {
		parent.Snapshots = map[string]state.SnapshotState{}
	}
	parent.Snapshots[seedName] = state.SnapshotState{
		Name:      seedName,
		CreatedAt: time.Now().UTC(),
		Kind:      "overlay",
		Dir:       snapDir,
		DiskPath:  base,
	}
	currentState.Instances[options.Target] = parent
	return base, nil
}

func nextForkName(current state.State, parent string) string {
	for n := 1; ; n++ {
		name := fmt.Sprintf("%s-fork-%d", parent, n)
		if _, exists := current.Instances[name]; !exists {
			return name
		}
	}
}

// allocFreeHostPort finds the first unused host port >= start on host,
// probing the host TCP stack so forks never collide with external processes.
func allocFreeHostPort(used map[string]string, host string, start int) int {
	if host == "" {
		host = "127.0.0.1"
	}
	port := start
	for port < 65535 {
		if _, taken := used[hostPortKey(host, port)]; !taken {
			ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
			if err == nil {
				_ = ln.Close()
				return port
			}
		}
		port++
	}
	return start
}

// reallocPortForwards copies the parent's forwards onto fresh host ports.
func reallocPortForwards(forwards []rtm.PortForwardPlan, used map[string]string, name string) []rtm.PortForwardPlan {
	if len(forwards) == 0 {
		return nil
	}
	out := make([]rtm.PortForwardPlan, 0, len(forwards))
	for _, fw := range forwards {
		fw.HostPort = allocFreeHostPort(used, fw.Host, fw.HostPort+1)
		used[hostPortKey(fw.Host, fw.HostPort)] = name + " port"
		out = append(out, fw)
	}
	return out
}

func statePortForwardsFromPlan(forwards []rtm.PortForwardPlan) []state.PortForwardState {
	if len(forwards) == 0 {
		return nil
	}
	out := make([]state.PortForwardState, 0, len(forwards))
	for _, fw := range forwards {
		out = append(out, state.PortForwardState{
			Name: fw.Name, Host: fw.Host, HostPort: fw.HostPort, GuestPort: fw.GuestPort, Protocol: fw.Protocol,
		})
	}
	return out
}

func copySmallFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

// DestroyForks removes every fork of the given instance (the parent survives).
func (s *Service) DestroyForks(ctx context.Context, options DestroyOptions, parent string) (DestroyResult, error) {
	root := options.ProjectRoot
	if root == "" {
		root = "."
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return DestroyResult{}, WrapError(ErrorCodeInternal, fmt.Sprintf("resolve project root: %v", err), err)
	}
	metadata, err := project.LoadMetadata(absoluteRoot)
	if err != nil {
		if errors.Is(err, project.ErrMetadataNotFound) {
			return DestroyResult{}, WrapError(ErrorCodePrecondition, err.Error(), err)
		}
		return DestroyResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	yeastHome, err := s.resolveYeastHome()
	if err != nil {
		return DestroyResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	paths, err := project.NewPaths(yeastHome, metadata)
	if err != nil {
		return DestroyResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	lock, err := state.Acquire(paths.StateLock, state.DefaultLockOptions())
	if err != nil {
		return DestroyResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	defer func() { _ = lock.Release() }()
	currentState, err := state.Load(paths.StateFile, metadata.ID)
	if err != nil {
		return DestroyResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}

	result := DestroyResult{ProjectID: metadata.ID}
	for _, name := range sortedInstanceNames(currentState) {
		inst := currentState.Instances[name]
		if inst.ForkOf != parent {
			continue
		}
		if err := s.runtime.Destroy(ctx, rtm.RuntimeInstance{
			Name: name, RuntimeDir: inst.RuntimeDir, PID: inst.PID,
		}); err != nil {
			return DestroyResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
		}
		delete(currentState.Instances, name)
		result.Instances = append(result.Instances, DestroyInstanceResult{Name: name, Status: "destroyed"})
	}
	if len(result.Instances) == 0 {
		return DestroyResult{}, WrapError(ErrorCodeNotFound, fmt.Sprintf("no forks found for instance %q", parent), nil)
	}
	if err := state.Save(paths.StateFile, currentState); err != nil {
		return DestroyResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	return result, nil
}
