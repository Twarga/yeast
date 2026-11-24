package app

import (
	"context"
	"errors"
	"fmt"
	"github.com/Twarga/yeast/internal/config"
	"github.com/Twarga/yeast/internal/guest"
	"github.com/Twarga/yeast/internal/project"
	provssh "github.com/Twarga/yeast/internal/provision/ssh"
	rtm "github.com/Twarga/yeast/internal/runtime"
	"github.com/Twarga/yeast/internal/state"
	"path/filepath"
	"strings"
	"time"
)

type RestoreOptions struct {
	ProjectRoot string
	Target      string
	Name        string
	Events      EventSink
}

type RestoreResult struct {
	ProjectID string              `json:"project_id"`
	Instance  string              `json:"instance"`
	Snapshot  state.SnapshotState `json:"snapshot"`
}

func (s *Service) Restore(ctx context.Context, options RestoreOptions) (out RestoreResult, err error) {
	started := time.Now()
	root := options.ProjectRoot
	if root == "" {
		root = "."
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return RestoreResult{}, WrapError(ErrorCodeInternal, fmt.Sprintf("resolve project root: %v", err), err)
	}
	defer func() {
		s.auditCall(ctx, absoluteRoot, "restore", map[string]any{"instance": options.Target, "snapshot": options.Name}, started, err)
	}()
	if strings.TrimSpace(options.Target) == "" {
		return RestoreResult{}, WrapError(ErrorCodeInvalidArgument, "restore target instance is required", nil)
	}
	if !isValidSnapshotName(options.Name) {
		return RestoreResult{}, WrapError(ErrorCodeInvalidArgument, fmt.Sprintf("invalid snapshot name %q", options.Name), nil)
	}

	metadata, err := project.LoadMetadata(absoluteRoot)
	if err != nil {
		if errors.Is(err, project.ErrMetadataNotFound) {
			return RestoreResult{}, WrapError(ErrorCodePrecondition, err.Error(), err)
		}
		return RestoreResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	emitEvent(options.Events, "restore", EventProjectLoaded, EventOptions{
		ProjectID: metadata.ID,
		Message:   "Project metadata loaded",
	})
	yeastHome, err := s.resolveYeastHome()
	if err != nil {
		return RestoreResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	paths, err := project.NewPaths(yeastHome, metadata)
	if err != nil {
		return RestoreResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}

	lock, err := state.Acquire(paths.StateLock, state.DefaultLockOptions())
	if err != nil {
		return RestoreResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	defer func() { _ = lock.Release() }()

	currentState, err := state.Load(paths.StateFile, metadata.ID)
	if err != nil {
		return RestoreResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}

	instance, ok := currentState.Instances[options.Target]
	if !ok {
		return RestoreResult{}, WrapError(ErrorCodeNotFound, fmt.Sprintf("instance %q not found", options.Target), nil)
	}
	snapshot, ok := instance.Snapshots[options.Name]
	if !ok {
		return RestoreResult{}, WrapError(ErrorCodeNotFound, fmt.Sprintf("snapshot %q not found for instance %q", options.Name, options.Target), nil)
	}
	if snapshot.Kind == "live" {
		return s.restoreLive(ctx, metadata, paths, currentState, instance, snapshot, options)
	}
	if instance.Status != "stopped" {
		return RestoreResult{}, WrapError(ErrorCodePrecondition, fmt.Sprintf("instance %q must be stopped before restore", options.Target), nil)
	}
	if instance.RuntimeDir == "" {
		return RestoreResult{}, WrapError(ErrorCodeInternal, fmt.Sprintf("instance %q runtime directory is missing", options.Target), nil)
	}

	diskPath := activeDiskPath(instance)
	emitEvent(options.Events, "restore", EventRestoreStarted, EventOptions{
		ProjectID: metadata.ID,
		Instance:  options.Target,
		Message:   "Restore started",
		Data: map[string]any{
			"snapshot":  options.Name,
			"disk_path": diskPath,
		},
	})
	kind := snapshot.Kind
	if kind == "" {
		kind = "copy"
	}
	if err := s.runtime.RestoreSnapshot(ctx, rtm.SnapshotPlan{
		InstanceDiskPath: diskPath,
		SnapshotPath:     snapshot.DiskPath,
		Kind:             kind,
		RuntimeDir:       instance.RuntimeDir,
	}); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return RestoreResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
		}
		if strings.Contains(err.Error(), "inspect snapshot file") {
			return RestoreResult{}, WrapError(ErrorCodeNotFound, err.Error(), err)
		}
		return RestoreResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}
	instance.ProvisioningStatus = state.ProvisioningStatusReady
	// Restored overlay is the new conventional active disk.
	instance.ActiveDisk = ""
	currentState.Instances[options.Target] = instance
	if err := state.Save(paths.StateFile, currentState); err != nil {
		return RestoreResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}

	emitEvent(options.Events, "restore", EventRestoreFinished, EventOptions{
		ProjectID: metadata.ID,
		Instance:  options.Target,
		Message:   "Restore finished",
		Data: map[string]any{
			"snapshot": options.Name,
		},
	})
	emitEvent(options.Events, "restore", EventWorkflowCompleted, EventOptions{
		ProjectID: metadata.ID,
		Instance:  options.Target,
		Message:   "Workflow completed",
	})

	return RestoreResult{
		ProjectID: metadata.ID,
		Instance:  options.Target,
		Snapshot:  snapshot,
	}, nil
}

// restoreLive respawns a running-state instance from a RAM+disk checkpoint.
// Unlike overlay restore it works whether the VM is stopped or running — a
// live guest is stopped cleanly first, then QEMU is respawned with
// -incoming defer and fed the saved memory image.
func (s *Service) restoreLive(ctx context.Context, metadata project.Metadata, paths project.Paths, currentState state.State, instance state.InstanceState, snapshot state.SnapshotState, options RestoreOptions) (RestoreResult, error) {
	restorer, ok := s.runtime.(rtm.LiveSnapshotter)
	if !ok {
		return RestoreResult{}, WrapError(ErrorCodePrecondition, "this runtime cannot restore live snapshots", nil)
	}
	if instance.RuntimeDir == "" {
		return RestoreResult{}, WrapError(ErrorCodeInternal, fmt.Sprintf("instance %q runtime directory is missing", options.Target), nil)
	}

	// Stop the running guest first (graceful powerdown → SIGTERM → SIGKILL).
	if instance.Status == "running" && instance.PID > 0 {
		rt := instanceToRuntime(options.Target, instance)
		if err := s.runtime.Stop(ctx, rt, 45*time.Second); err != nil {
			return RestoreResult{}, WrapError(ErrorCodeInternal, fmt.Sprintf("stop running instance: %v", err), err)
		}
	}

	emitEvent(options.Events, "restore", EventRestoreStarted, EventOptions{
		ProjectID: metadata.ID,
		Instance:  options.Target,
		Message:   "Live restore started",
		Data:      map[string]any{"snapshot": options.Name},
	})

	fresh, err := restorer.RestoreLiveSnapshot(ctx, rtm.LiveRestorePlan{
		Instance:    instanceToRuntime(options.Target, instance),
		SnapshotDir: snapshot.Dir,
	})
	if err != nil {
		return RestoreResult{}, WrapError(ErrorCodeInternal, fmt.Sprintf("restore live snapshot: %v", err), err)
	}

	instance.PID = fresh.PID
	instance.Status = "running"
	instance.ActiveDisk = ""
	if fresh.Display != nil {
		instance.Display = &state.DisplayState{
			Type: fresh.Display.Type, Host: fresh.Display.Host, Port: fresh.Display.Port,
			URL: fresh.Display.URL, GPU: fresh.Display.GPU, Firmware: fresh.Display.Firmware,
		}
	}
	currentState.Instances[options.Target] = instance
	if err := state.Save(paths.StateFile, currentState); err != nil {
		return RestoreResult{}, WrapError(ErrorCodeInternal, err.Error(), err)
	}

	// Resync the guest clock — the restored RAM image carries a frozen
	// timestamp; TLS, apt/pacman and logs break without this.
	s.resyncGuestClock(ctx, options.ProjectRoot, metadata.ID, options.Target, instance, options.Events)

	emitEvent(options.Events, "restore", EventRestoreFinished, EventOptions{
		ProjectID: metadata.ID,
		Instance:  options.Target,
		Message:   "Live restore finished",
		Data:      map[string]any{"snapshot": options.Name},
	})
	emitEvent(options.Events, "restore", EventWorkflowCompleted, EventOptions{
		ProjectID: metadata.ID,
		Instance:  options.Target,
		Message:   "Workflow completed",
	})
	return RestoreResult{ProjectID: metadata.ID, Instance: options.Target, Snapshot: snapshot}, nil
}

// resyncGuestClock waits for SSH then steps the guest clock to host time.
// Best-effort: failures surface as a warning event, not a failed restore.
func (s *Service) resyncGuestClock(ctx context.Context, projectRoot, projectID, name string, instance state.InstanceState, events EventSink) {
	if instance.SSHPort <= 0 {
		return
	}
	cfg, _ := config.Load(filepath.Join(projectRoot, ConfigFileName))
	host := resolveManagementHost(cfg)
	address, err := s.sshAddress(host, instance.SSHPort)
	if err != nil {
		return
	}
	if err := s.waitForTCP(ctx, guest.ReadinessOptions{Address: address, Timeout: defaultReadinessTimeout}); err != nil {
		return
	}
	transport := s.provisionTransport
	if transport == nil {
		transport = provssh.NewLocalTransport()
	}
	user := instance.User
	if user == "" {
		user = "yeast"
	}
	now := time.Now().UTC().Unix()
	cmd := fmt.Sprintf("sudo date -u -s '@%d' >/dev/null 2>&1 || sudo chronyc makestep >/dev/null 2>&1", now)
	if res, runErr := transport.Run(ctx, provssh.RunRequest{User: user, Host: host, Port: instance.SSHPort, Command: cmd, Timeout: 30 * time.Second}); runErr == nil && res.ExitCode == 0 {
		emitEvent(events, "restore", EventClockResynced, EventOptions{
			ProjectID: projectID,
			Instance:  name,
			Message:   "Guest clock resynced",
		})
	}
}
