package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Twarga/yeast/internal/project"
	rtm "github.com/Twarga/yeast/internal/runtime"
	"github.com/Twarga/yeast/internal/state"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type SnapshotOptions struct {
	ProjectRoot string
	Target      string
	Name        string
	Description string
	Events      EventSink
}

type SnapshotResult struct {
	ProjectID string              `json:"project_id"`
	Instance  string              `json:"instance"`
	Snapshot  state.SnapshotState `json:"snapshot"`
}

func (s *Service) Snapshot(ctx context.Context, options SnapshotOptions) (out SnapshotResult, err error) {
	started := time.Now()
	root := options.ProjectRoot
	if root == "" {
		root = "."
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return SnapshotResult{}, WrapError(ErrorCodeInternal, fmt.Sprintf("resolve project root: %v", err), err)
	}
	defer func() {
		s.auditCall(ctx, absoluteRoot, "snapshot", map[string]any{"instance": options.Target, "snapshot": options.Name}, started, err)
	}()
	if strings.TrimSpace(options.Target) == "" {
		return SnapshotResult{}, WrapError(ErrorCodeInvalidArgument, "snapshot target instance is required", nil)
	}
	if !isValidSnapshotName(options.Name) {
		return SnapshotResult{}, WrapError(ErrorCodeInvalidArgument, fmt.Sprintf("invalid snapshot name %q", options.Name), nil)
	}

	metadata, err := project.LoadMetadata(absoluteRoot)
	if err != nil {
		if errors.Is(err, project.ErrMetadataNotFound) {
			return SnapshotResult{}, WrapError(ErrorCodePrecondition, err.Error(), err)
		}
		return SnapshotResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	yeastHome, err := s.resolveYeastHome()
	if err != nil {
		return SnapshotResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	paths, err := project.NewPaths(yeastHome, metadata)
	if err != nil {
		return SnapshotResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}

	lock, err := state.Acquire(paths.StateLock, state.DefaultLockOptions())
	if err != nil {
		return SnapshotResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	defer func() { _ = lock.Release() }()

	currentState, err := state.Load(paths.StateFile, metadata.ID)
	if err != nil {
		return SnapshotResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}

	instance, ok := currentState.Instances[options.Target]
	if !ok {
		return SnapshotResult{}, WrapError(ErrorCodeNotFound, fmt.Sprintf("instance %q not found", options.Target), nil)
	}
	if instance.Status != "stopped" {
		return s.snapshotLive(ctx, metadata, paths, instance, options)
	}
	if instance.RuntimeDir == "" {
		return SnapshotResult{}, WrapError(ErrorCodeInternal, fmt.Sprintf("instance %q runtime directory is missing", options.Target), nil)
	}
	if _, exists := instance.Snapshots[options.Name]; exists {
		return SnapshotResult{}, WrapError(ErrorCodeConflict, fmt.Sprintf("snapshot %q already exists for instance %q", options.Name, options.Target), nil)
	}

	snapshotDir, err := paths.SnapshotDir(options.Target)
	if err != nil {
		return SnapshotResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	// Overlay snapshots store their frozen disk as <snapDir>/base.qcow2.
	snapDir := filepath.Join(snapshotDir, options.Name)
	snapshotPath := filepath.Join(snapDir, "base.qcow2")
	diskPath := activeDiskPath(instance)
	if err := s.runtime.CreateSnapshot(ctx, rtm.SnapshotPlan{
		InstanceDiskPath: diskPath,
		SnapshotPath:     snapshotPath,
		Kind:             "overlay",
		RuntimeDir:       instance.RuntimeDir,
	}); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return SnapshotResult{}, WrapError(ErrorCodeConflict, err.Error(), err)
		}
		return SnapshotResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}

	snapshot := state.SnapshotState{
		Name:           options.Name,
		CreatedAt:      time.Now().UTC(),
		Description:    strings.TrimSpace(options.Description),
		Kind:           "overlay",
		DiskPath:       snapshotPath,
		Dir:            snapDir,
		SourceDiskSize: snapshotSourceDiskSize(snapshotPath),
	}
	if instance.Snapshots == nil {
		instance.Snapshots = make(map[string]state.SnapshotState)
	}
	instance.Snapshots[options.Name] = snapshot
	// The new active layer is always <RuntimeDir>/disk.qcow2 after an overlay
	// snapshot — clear any external ActiveDisk pointer.
	instance.ActiveDisk = ""
	currentState.Instances[options.Target] = instance
	if err := state.Save(paths.StateFile, currentState); err != nil {
		return SnapshotResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}

	emitEvent(options.Events, "snapshot", EventSnapshotCreated, EventOptions{
		ProjectID: metadata.ID,
		Instance:  options.Target,
		Message:   fmt.Sprintf("Snapshot %s created", options.Name),
	})

	return SnapshotResult{
		ProjectID: metadata.ID,
		Instance:  options.Target,
		Snapshot:  snapshot,
	}, nil
}

const bootPlanName = "machine-plan.json"

// writeBootPlan persists the MachinePlan used to boot an instance so live
// snapshots and forks can respawn with identical hardware topology.
func writeBootPlan(runtimeDir string, plan rtm.MachinePlan) error {
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(runtimeDir, bootPlanName), data, 0644)
}

func loadBootPlan(runtimeDir string) (rtm.MachinePlan, error) {
	var plan rtm.MachinePlan
	data, err := os.ReadFile(filepath.Join(runtimeDir, bootPlanName))
	if err != nil {
		return plan, err
	}
	if err := json.Unmarshal(data, &plan); err != nil {
		return plan, fmt.Errorf("parse machine plan: %w", err)
	}
	return plan, nil
}

// instanceToRuntime builds the runtime view of a state instance.
func instanceToRuntime(name string, instance state.InstanceState) rtm.RuntimeInstance {
	out := rtm.RuntimeInstance{
		Name:       name,
		RuntimeDir: instance.RuntimeDir,
		PID:        instance.PID,
		ActiveDisk: instance.ActiveDisk,
	}
	if instance.Display != nil {
		out.Display = &rtm.DisplayInfo{
			Type: instance.Display.Type, Host: instance.Display.Host, Port: instance.Display.Port,
			URL: instance.Display.URL, GPU: instance.Display.GPU, Firmware: instance.Display.Firmware,
		}
	}
	return out
}

// activeDiskPath returns the file QEMU currently writes to for an instance.
func activeDiskPath(instance state.InstanceState) string {
	if instance.ActiveDisk != "" {
		return instance.ActiveDisk
	}
	return filepath.Join(instance.RuntimeDir, "disk.qcow2")
}

// snapshotLive checkpoints a running instance (RAM + disk) via QMP.
func (s *Service) snapshotLive(ctx context.Context, metadata project.Metadata, paths project.Paths, instance state.InstanceState, options SnapshotOptions) (SnapshotResult, error) {
	snapper, ok := s.runtime.(rtm.LiveSnapshotter)
	if !ok {
		return SnapshotResult{}, WrapError(ErrorCodePrecondition, fmt.Sprintf("instance %q is running and this runtime cannot do live snapshots (stop it first)", options.Target), nil)
	}
	if _, exists := instance.Snapshots[options.Name]; exists {
		return SnapshotResult{}, WrapError(ErrorCodeConflict, fmt.Sprintf("snapshot %q already exists for instance %q", options.Name, options.Target), nil)
	}
	snapshotDir, err := paths.SnapshotDir(options.Target)
	if err != nil {
		return SnapshotResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	snapDir := filepath.Join(snapshotDir, options.Name)
	machinePlan, err := loadBootPlan(instance.RuntimeDir)
	if err != nil {
		return SnapshotResult{}, WrapError(ErrorCodePrecondition, fmt.Sprintf("instance %q has no recorded machine plan — run 'yeast down %s && yeast up' once to enable live snapshots (%v)", options.Target, options.Target, err), nil)
	}
	info, err := snapper.SaveLiveSnapshot(ctx, rtm.LiveSnapshotPlan{
		Instance:    instanceToRuntime(options.Target, instance),
		SnapshotDir: snapDir,
		MemoryMiB:   machinePlan.MemoryMiB,
		Plan:        machinePlan,
	})
	if err != nil {
		return SnapshotResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}

	snapshot := state.SnapshotState{
		Name:        options.Name,
		CreatedAt:   time.Now().UTC(),
		Description: strings.TrimSpace(options.Description),
		Kind:        "live",
		Dir:         snapDir,
		DiskPath:    filepath.Join(snapDir, "base.qcow2"),
		MemoryPath:  filepath.Join(snapDir, "memory.bin"),
		Method:      info.Method,
		PauseMs:     info.PauseMs,
	}
	if instance.Snapshots == nil {
		instance.Snapshots = make(map[string]state.SnapshotState)
	}
	instance.Snapshots[options.Name] = snapshot
	// The running guest now writes into the overlay inside the snapshot dir.
	instance.ActiveDisk = filepath.Join(snapDir, "overlay.qcow2")

	currentState, err := state.Load(paths.StateFile, metadata.ID)
	if err != nil {
		return SnapshotResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	currentState.Instances[options.Target] = instance
	if err := state.Save(paths.StateFile, currentState); err != nil {
		return SnapshotResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}

	emitEvent(options.Events, "snapshot", EventSnapshotCreated, EventOptions{
		ProjectID: metadata.ID,
		Instance:  options.Target,
		Message:   fmt.Sprintf("Live snapshot %s created (%s, %dms pause)", options.Name, info.Method, info.PauseMs),
	})
	return SnapshotResult{ProjectID: metadata.ID, Instance: options.Target, Snapshot: snapshot}, nil
}

func isValidSnapshotName(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	if name == "." || name == ".." {
		return false
	}
	if strings.ContainsRune(name, filepath.Separator) {
		return false
	}
	return !strings.Contains(name, "..")
}

func snapshotSourceDiskSize(diskPath string) string {
	info, err := os.Stat(diskPath)
	if err != nil || info.Size() <= 0 {
		return ""
	}
	return fmt.Sprintf("%d", info.Size())
}
