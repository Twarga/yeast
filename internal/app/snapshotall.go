package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/Twarga/yeast/internal/project"
	rtm "github.com/Twarga/yeast/internal/runtime"
	"github.com/Twarga/yeast/internal/state"
)

// safetySnapshotName is the internal checkpoint taken before `restore --all`
// so a mid-way failure can roll back already-restored instances.
const safetySnapshotName = "__pre-restore"

type SnapshotAllOptions struct {
	ProjectRoot string
	Name        string
	Description string
	Events      EventSink
}

type SnapshotAllEntry struct {
	Instance string              `json:"instance"`
	Snapshot state.SnapshotState `json:"snapshot"`
}

type SnapshotAllResult struct {
	ProjectID string             `json:"project_id"`
	Name      string             `json:"name"`
	Snapshots []SnapshotAllEntry `json:"snapshots"`
}

// SnapshotAll checkpoints every state instance under one shared name.
// On any failure the snapshots already created are deleted (transactional).
func (s *Service) SnapshotAll(ctx context.Context, options SnapshotAllOptions) (SnapshotAllResult, error) {
	if !isValidSnapshotName(options.Name) {
		return SnapshotAllResult{}, WrapError(ErrorCodeInvalidArgument, fmt.Sprintf("invalid snapshot name %q", options.Name), nil)
	}
	metadata, paths, currentState, done, err := s.snapshotAllSetup(options.ProjectRoot)
	if err != nil {
		return SnapshotAllResult{}, err
	}
	defer done()

	// Preflight: no instance may already have this name.
	for name, inst := range currentState.Instances {
		if _, exists := inst.Snapshots[options.Name]; exists {
			return SnapshotAllResult{}, WrapError(ErrorCodeConflict, fmt.Sprintf("snapshot %q already exists for instance %q", options.Name, name), nil)
		}
	}

	created := []SnapshotAllEntry{}
	for _, name := range sortedInstanceNames(currentState) {
		res, err := s.snapshotOne(ctx, metadata, paths, &currentState, name, options.Name, options.Description, options.Events)
		if err != nil {
			// rollback
			for _, c := range created {
				inst := currentState.Instances[c.Instance]
				_ = s.runtime.DeleteSnapshot(ctx, rtm.DeleteSnapshotPlan{
					Path: c.Snapshot.DiskPath, Kind: c.Snapshot.Kind,
					RuntimeDir: inst.RuntimeDir, ActiveDiskPath: inst.ActiveDisk,
				})
				delete(inst.Snapshots, options.Name)
				currentState.Instances[c.Instance] = inst
			}
			_ = state.Save(paths.StateFile, currentState)
			return SnapshotAllResult{}, err
		}
		created = append(created, SnapshotAllEntry{Instance: name, Snapshot: res})
	}
	if err := state.Save(paths.StateFile, currentState); err != nil {
		return SnapshotAllResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	return SnapshotAllResult{ProjectID: metadata.ID, Name: options.Name, Snapshots: created}, nil
}

// snapshotOne runs the per-instance create (overlay or live) against shared
// state without re-acquiring the lock.
func (s *Service) snapshotOne(ctx context.Context, metadata project.Metadata, paths project.Paths, currentState *state.State, name, snapName, description string, events EventSink) (state.SnapshotState, error) {
	instance := currentState.Instances[name]
	snapshotDir, err := paths.SnapshotDir(name)
	if err != nil {
		return state.SnapshotState{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	snapDir := filepath.Join(snapshotDir, snapName)
	base := filepath.Join(snapDir, "base.qcow2")

	if instance.Status == "running" {
		snapper, ok := s.runtime.(rtm.LiveSnapshotter)
		if !ok {
			return state.SnapshotState{}, WrapError(ErrorCodePrecondition, fmt.Sprintf("instance %q is running and this runtime cannot do live snapshots", name), nil)
		}
		plan, err := loadBootPlan(instance.RuntimeDir)
		if err != nil {
			return state.SnapshotState{}, WrapError(ErrorCodePrecondition, fmt.Sprintf("instance %q has no recorded machine plan (%v)", name, err), nil)
		}
		info, err := snapper.SaveLiveSnapshot(ctx, rtm.LiveSnapshotPlan{
			Instance: instanceToRuntime(name, instance), SnapshotDir: snapDir,
			MemoryMiB: plan.MemoryMiB, Plan: plan,
		})
		if err != nil {
			return state.SnapshotState{}, WrapError(ErrorCodeInternal, err.Error(), err)
		}
		snap := state.SnapshotState{
			Name: snapName, CreatedAt: time.Now().UTC(), Description: description,
			Kind: "live", Dir: snapDir, DiskPath: base,
			MemoryPath: filepath.Join(snapDir, "memory.bin"),
			Method:     info.Method, PauseMs: info.PauseMs,
		}
		instance.ActiveDisk = filepath.Join(snapDir, "overlay.qcow2")
		if instance.Snapshots == nil {
			instance.Snapshots = map[string]state.SnapshotState{}
		}
		instance.Snapshots[snapName] = snap
		currentState.Instances[name] = instance
		return snap, nil
	}

	if err := s.runtime.CreateSnapshot(ctx, rtm.SnapshotPlan{
		InstanceDiskPath: activeDiskPath(instance), SnapshotPath: base,
		Kind: "overlay", RuntimeDir: instance.RuntimeDir,
	}); err != nil {
		if errors.Is(err, context.Canceled) {
			return state.SnapshotState{}, WrapError(ErrorCodeInternal, err.Error(), err)
		}
		return state.SnapshotState{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	snap := state.SnapshotState{
		Name: snapName, CreatedAt: time.Now().UTC(), Description: description,
		Kind: "overlay", Dir: snapDir, DiskPath: base,
		SourceDiskSize: snapshotSourceDiskSize(base),
	}
	instance.ActiveDisk = ""
	if instance.Snapshots == nil {
		instance.Snapshots = map[string]state.SnapshotState{}
	}
	instance.Snapshots[snapName] = snap
	currentState.Instances[name] = instance
	return snap, nil
}

// --- restore --all ------------------------------------------------------------

type RestoreAllOptions struct {
	ProjectRoot string
	Name        string
	Events      EventSink
}

type RestoreAllResult struct {
	ProjectID string   `json:"project_id"`
	Name      string   `json:"name"`
	Instances []string `json:"instances"`
}

// RestoreAll restores one shared checkpoint name across every instance.
// Before touching anything it (a) verifies all instances have the snapshot
// and (b) takes a safety overlay snapshot so a mid-way failure rolls back.
func (s *Service) RestoreAll(ctx context.Context, options RestoreAllOptions) (RestoreAllResult, error) {
	if !isValidSnapshotName(options.Name) {
		return RestoreAllResult{}, WrapError(ErrorCodeInvalidArgument, fmt.Sprintf("invalid snapshot name %q", options.Name), nil)
	}
	metadata, paths, currentState, done, err := s.snapshotAllSetup(options.ProjectRoot)
	if err != nil {
		return RestoreAllResult{}, err
	}
	defer done()

	// Preflight: all snapshots must exist; stopped-kind snapshots require a
	// stopped instance (live restores handle running guests themselves).
	for _, name := range sortedInstanceNames(currentState) {
		inst := currentState.Instances[name]
		snap, ok := inst.Snapshots[options.Name]
		if !ok {
			return RestoreAllResult{}, WrapError(ErrorCodeNotFound, fmt.Sprintf("instance %q has no snapshot %q — aborting before any restore", name, options.Name), nil)
		}
		if snap.Kind != "live" && inst.Status != "stopped" {
			return RestoreAllResult{}, WrapError(ErrorCodePrecondition, fmt.Sprintf("instance %q is running with a stopped-kind snapshot %q — stop it first (nothing restored)", name, options.Name), nil)
		}
	}

	names := sortedInstanceNames(currentState)
	safetyCreated := map[string]bool{}
	var restored []string
	for _, name := range names {
		inst := currentState.Instances[name]
		snap := inst.Snapshots[options.Name]

		// Safety checkpoint for rollback (stopped instances only — live
		// restores already replace state atomically per instance).
		haveSafety := false
		if inst.Status == "stopped" {
			snapDir, _ := paths.SnapshotDir(name)
			safetyDir := filepath.Join(snapDir, safetySnapshotName)
			_ = os.RemoveAll(safetyDir)
			if err := s.runtime.CreateSnapshot(ctx, rtm.SnapshotPlan{
				InstanceDiskPath: activeDiskPath(inst),
				SnapshotPath:     filepath.Join(safetyDir, "base.qcow2"),
				Kind:             "overlay",
				RuntimeDir:       inst.RuntimeDir,
			}); err != nil {
				s.rollbackRestored(ctx, paths, &currentState, restored, safetySnapshotName)
				s.deleteSafetySnapshots(ctx, paths, &currentState, safetyCreated)
				_ = state.Save(paths.StateFile, currentState)
				return RestoreAllResult{}, WrapError(ErrorCodeInternal, fmt.Sprintf("safety checkpoint for %s: %v", name, err), err)
			}
			haveSafety = true
			safetyCreated[name] = true
		}

		var restoreErr error
		if snap.Kind == "live" {
			var restoredInst rtm.RuntimeInstance
			restoredInst, restoreErr = s.restoreLiveInstance(ctx, name, inst, snap)
			if restoreErr == nil {
				inst.PID = restoredInst.PID
				inst.Status = "running"
				inst.ActiveDisk = ""
				if restoredInst.Display != nil {
					inst.Display = &state.DisplayState{
						Type:     restoredInst.Display.Type,
						Host:     restoredInst.Display.Host,
						Port:     restoredInst.Display.Port,
						URL:      restoredInst.Display.URL,
						GPU:      restoredInst.Display.GPU,
						Firmware: restoredInst.Display.Firmware,
					}
				}
				currentState.Instances[name] = inst
				s.resyncGuestClock(ctx, options.ProjectRoot, metadata.ID, name, inst, options.Events)
			}
		} else {
			kind := snap.Kind
			if kind == "" {
				kind = "copy"
			}
			restoreErr = s.runtime.RestoreSnapshot(ctx, rtm.SnapshotPlan{
				InstanceDiskPath: activeDiskPath(inst), SnapshotPath: snap.DiskPath,
				Kind: kind, RuntimeDir: inst.RuntimeDir,
			})
			if restoreErr == nil {
				inst.ActiveDisk = ""
				inst.ProvisioningStatus = state.ProvisioningStatusReady
				currentState.Instances[name] = inst
			}
		}
		if restoreErr != nil {
			s.rollbackRestored(ctx, paths, &currentState, restored, safetySnapshotName)
			// also roll back THIS instance's attempted state via its safety
			if haveSafety {
				inst := currentState.Instances[name]
				_ = s.runtime.RestoreSnapshot(ctx, rtm.SnapshotPlan{
					InstanceDiskPath: filepath.Join(inst.RuntimeDir, "disk.qcow2"),
					SnapshotPath:     filepath.Join(snapDirOf(paths, name), safetySnapshotName, "base.qcow2"),
					Kind:             "overlay", RuntimeDir: inst.RuntimeDir,
				})
			}
			s.deleteSafetySnapshots(ctx, paths, &currentState, safetyCreated)
			_ = state.Save(paths.StateFile, currentState)
			return RestoreAllResult{}, WrapError(ErrorCodeInternal, fmt.Sprintf("restore %s: %v (all restored instances rolled back)", name, restoreErr), restoreErr)
		}
		restored = append(restored, name)
	}

	s.deleteSafetySnapshots(ctx, paths, &currentState, safetyCreated)
	if err := state.Save(paths.StateFile, currentState); err != nil {
		return RestoreAllResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	emitEvent(options.Events, "restore", EventRestoreFinished, EventOptions{
		ProjectID: metadata.ID, Message: fmt.Sprintf("Restored %d instance(s) from %q", len(restored), options.Name),
	})
	return RestoreAllResult{ProjectID: metadata.ID, Name: options.Name, Instances: restored}, nil
}

// restoreLiveInstance stops a running guest then respawns from the checkpoint.
func (s *Service) restoreLiveInstance(ctx context.Context, name string, inst state.InstanceState, snap state.SnapshotState) (rtm.RuntimeInstance, error) {
	restorer, ok := s.runtime.(rtm.LiveSnapshotter)
	if !ok {
		return rtm.RuntimeInstance{}, fmt.Errorf("runtime cannot restore live snapshots")
	}
	if inst.Status == "running" && inst.PID > 0 {
		if err := s.runtime.Stop(ctx, instanceToRuntime(name, inst), 45*time.Second); err != nil {
			return rtm.RuntimeInstance{}, err
		}
	}
	return restorer.RestoreLiveSnapshot(ctx, rtm.LiveRestorePlan{
		Instance: instanceToRuntime(name, inst), SnapshotDir: snap.Dir,
	})
}

// rollbackRestored re-applies the safety checkpoint on already-restored
// instances and removes all safety checkpoints.
func (s *Service) rollbackRestored(ctx context.Context, paths project.Paths, currentState *state.State, restored []string, safety string) {
	for _, name := range restored {
		inst := currentState.Instances[name]
		safetyBase := filepath.Join(snapDirOf(paths, name), safety, "base.qcow2")
		if _, err := os.Stat(safetyBase); err != nil {
			continue
		}
		_ = s.runtime.RestoreSnapshot(ctx, rtm.SnapshotPlan{
			InstanceDiskPath: filepath.Join(inst.RuntimeDir, "disk.qcow2"),
			SnapshotPath:     safetyBase, Kind: "overlay", RuntimeDir: inst.RuntimeDir,
		})
	}
}

// deleteSafetySnapshots removes the per-instance __pre-restore checkpoints.
func (s *Service) deleteSafetySnapshots(ctx context.Context, paths project.Paths, currentState *state.State, created map[string]bool) {
	for _, name := range sortedInstanceNames(*currentState) {
		if !created[name] {
			continue
		}
		inst := currentState.Instances[name]
		dir := filepath.Join(snapDirOf(paths, name), safetySnapshotName)
		base := filepath.Join(dir, "base.qcow2")
		_ = os.Chmod(base, 0644)
		_ = s.runtime.DeleteSnapshot(ctx, rtm.DeleteSnapshotPlan{
			Path: base, Kind: "overlay", RuntimeDir: inst.RuntimeDir, ActiveDiskPath: inst.ActiveDisk,
		})
		delete(inst.Snapshots, safetySnapshotName)
		currentState.Instances[name] = inst
	}
}

// --- shared -----------------------------------------------------------------

func snapDirOf(paths project.Paths, name string) string {
	dir, err := paths.SnapshotDir(name)
	if err != nil {
		return ""
	}
	return dir
}

func sortedInstanceNames(currentState state.State) []string {
	names := make([]string, 0, len(currentState.Instances))
	for name := range currentState.Instances {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (s *Service) snapshotAllSetup(projectRoot string) (project.Metadata, project.Paths, state.State, func(), error) {
	root := projectRoot
	if root == "" {
		root = "."
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return project.Metadata{}, project.Paths{}, state.State{}, func() {}, WrapError(ErrorCodeInternal, fmt.Sprintf("resolve project root: %v", err), err)
	}
	metadata, err := project.LoadMetadata(absoluteRoot)
	if err != nil {
		if errors.Is(err, project.ErrMetadataNotFound) {
			return project.Metadata{}, project.Paths{}, state.State{}, func() {}, WrapError(ErrorCodePrecondition, err.Error(), err)
		}
		return project.Metadata{}, project.Paths{}, state.State{}, func() {}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	yeastHome, err := s.resolveYeastHome()
	if err != nil {
		return project.Metadata{}, project.Paths{}, state.State{}, func() {}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	paths, err := project.NewPaths(yeastHome, metadata)
	if err != nil {
		return project.Metadata{}, project.Paths{}, state.State{}, func() {}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	lock, err := state.Acquire(paths.StateLock, state.DefaultLockOptions())
	if err != nil {
		return project.Metadata{}, project.Paths{}, state.State{}, func() {}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	currentState, err := state.Load(paths.StateFile, metadata.ID)
	if err != nil {
		_ = lock.Release()
		return project.Metadata{}, project.Paths{}, state.State{}, func() {}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	return metadata, paths, currentState, func() { _ = lock.Release() }, nil
}
