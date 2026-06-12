package qemu

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
	rtm "yeast/internal/runtime"
)

type Runtime struct{}

func NewRuntime() *Runtime {
	return &Runtime{}
}

func (r *Runtime) PrepareDisk(ctx context.Context, plan rtm.MachinePlan) (rtm.DiskPlan, error) {
	return PrepareDisk(ctx, plan)
}

func (r *Runtime) Start(ctx context.Context, plan rtm.MachinePlan) (rtm.RuntimeInstance, error) {
	if plan.RuntimeDir == "" {
		return rtm.RuntimeInstance{}, fmt.Errorf("runtime directory is required")
	}
	if plan.LogPath == "" {
		return rtm.RuntimeInstance{}, fmt.Errorf("log path is required")
	}
	if err := os.MkdirAll(plan.RuntimeDir, 0755); err != nil {
		return rtm.RuntimeInstance{}, fmt.Errorf("create runtime directory %s: %w", plan.RuntimeDir, err)
	}
	if err := os.MkdirAll(filepath.Dir(plan.LogPath), 0755); err != nil {
		return rtm.RuntimeInstance{}, fmt.Errorf("create log directory for %s: %w", plan.LogPath, err)
	}
	if err := removeStaleQMPSocket(plan.RuntimeDir); err != nil {
		return rtm.RuntimeInstance{}, err
	}

	binary, args, err := buildCommand(plan)
	if err != nil {
		return rtm.RuntimeInstance{}, err
	}

	logFile, err := os.OpenFile(plan.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return rtm.RuntimeInstance{}, fmt.Errorf("open log file %s: %w", plan.LogPath, err)
	}
	defer func() {
		_ = logFile.Close()
	}()

	proc, err := startProcess(ctx, binary, args, logFile, logFile)
	if err != nil {
		return rtm.RuntimeInstance{}, fmt.Errorf("start qemu process: %w", err)
	}
	pid := proc.PID()
	if pid <= 0 {
		return rtm.RuntimeInstance{}, fmt.Errorf("qemu process returned invalid pid %d", pid)
	}
	if err := proc.Release(); err != nil {
		return rtm.RuntimeInstance{}, fmt.Errorf("release qemu process handle: %w", err)
	}

	return rtm.RuntimeInstance{
		Name:       plan.Name,
		RuntimeDir: plan.RuntimeDir,
		LogPath:    plan.LogPath,
		PID:        pid,
		Networks:   plan.Networks,
		StartedAt:  time.Now().UTC(),
	}, nil
}

func (r *Runtime) Stop(ctx context.Context, instance rtm.RuntimeInstance, timeout time.Duration) error {
	if instance.PID <= 0 {
		return fmt.Errorf("runtime instance pid is required")
	}

	running, err := processRunning(instance.PID)
	if err != nil {
		return fmt.Errorf("inspect process %d: %w", instance.PID, err)
	}
	if !running {
		return nil
	}

	qmpTimeout := timeout - 5*time.Second
	if qmpTimeout <= 0 {
		qmpTimeout = timeout
	}
	if qmpTimeout < 3*time.Second && timeout >= 3*time.Second {
		qmpTimeout = 3 * time.Second
	}
	if err := r.gracefulPowerdown(ctx, instance, qmpTimeout); err == nil {
		_ = removeStaleQMPSocket(instance.RuntimeDir)
		return nil
	} else {
		appendRuntimeLog(instance.RuntimeDir, fmt.Sprintf("yeast: graceful powerdown failed, falling back to SIGTERM: %v\n", err))
	}

	termTimeout := timeout - qmpTimeout
	if termTimeout < 0 {
		termTimeout = 0
	}
	if err := signalProcess(instance.PID, syscall.SIGTERM); err != nil && !isNoSuchProcess(err) {
		return fmt.Errorf("send SIGTERM to process %d: %w", instance.PID, err)
	}
	if err := waitForProcessExit(ctx, instance.PID, termTimeout); err == nil {
		return nil
	} else if err != context.DeadlineExceeded {
		return fmt.Errorf("wait for process %d after SIGTERM: %w", instance.PID, err)
	}

	if err := signalProcess(instance.PID, syscall.SIGKILL); err != nil && !isNoSuchProcess(err) {
		return fmt.Errorf("send SIGKILL to process %d: %w", instance.PID, err)
	}
	if err := waitForProcessExit(ctx, instance.PID, 5*time.Second); err != nil && err != context.DeadlineExceeded {
		return fmt.Errorf("wait for process %d after SIGKILL: %w", instance.PID, err)
	}
	_ = removeStaleQMPSocket(instance.RuntimeDir)

	return nil
}

func (r *Runtime) gracefulPowerdown(ctx context.Context, instance rtm.RuntimeInstance, timeout time.Duration) error {
	if instance.RuntimeDir == "" {
		return fmt.Errorf("runtime directory is required for graceful shutdown")
	}
	socketPath := qmpSocketPath(instance.RuntimeDir)
	if _, err := os.Stat(socketPath); err != nil {
		return fmt.Errorf("qmp socket not available: %w", err)
	}

	client, err := newQMPClient(socketPath, 2*time.Second)
	if err != nil {
		return fmt.Errorf("connect to qmp: %w", err)
	}
	defer client.close()

	if err := client.execute("system_powerdown", nil); err != nil {
		return fmt.Errorf("send system_powerdown: %w", err)
	}

	if err := waitForProcessExit(ctx, instance.PID, timeout); err != nil {
		return fmt.Errorf("wait for process exit after powerdown: %w", err)
	}
	return nil
}

func appendRuntimeLog(runtimeDir, message string) {
	if runtimeDir == "" {
		return
	}
	logPath := filepath.Join(runtimeDir, "vm.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	defer func() {
		_ = logFile.Close()
	}()
	_, _ = logFile.WriteString(message)
}

func (r *Runtime) Inspect(ctx context.Context, instance rtm.RuntimeInstance) (rtm.ProcessInfo, error) {
	_ = ctx
	if instance.PID <= 0 {
		return rtm.ProcessInfo{}, fmt.Errorf("runtime instance pid is required")
	}

	running, err := processRunning(instance.PID)
	if err != nil {
		return rtm.ProcessInfo{}, fmt.Errorf("inspect process %d: %w", instance.PID, err)
	}

	state := rtm.ProcessStateStopped
	if running {
		state = rtm.ProcessStateRunning
	}

	return rtm.ProcessInfo{
		PID:       instance.PID,
		State:     state,
		StartedAt: instance.StartedAt,
	}, nil
}

func (r *Runtime) CreateSnapshot(ctx context.Context, plan rtm.SnapshotPlan) error {
	return CreateSnapshotCopy(ctx, plan)
}

func (r *Runtime) RestoreSnapshot(ctx context.Context, plan rtm.SnapshotPlan) error {
	return RestoreSnapshotCopy(ctx, plan)
}

func (r *Runtime) DeleteSnapshot(ctx context.Context, snapshotPath string) error {
	return DeleteSnapshotFile(snapshotPath)
}

func (r *Runtime) Destroy(ctx context.Context, instance rtm.RuntimeInstance) error {
	if instance.PID > 0 {
		if err := r.Stop(ctx, instance, 5*time.Second); err != nil {
			running, inspectErr := processRunning(instance.PID)
			if inspectErr != nil || running {
				return err
			}
		}
	}
	if instance.RuntimeDir == "" {
		return nil
	}
	if err := os.RemoveAll(instance.RuntimeDir); err != nil {
		return fmt.Errorf("remove runtime directory %s: %w", instance.RuntimeDir, err)
	}
	return nil
}

func removeStaleQMPSocket(runtimeDir string) error {
	if runtimeDir == "" {
		return nil
	}
	socketPath := qmpSocketPath(runtimeDir)
	if err := os.Remove(socketPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove stale qmp socket %s: %w", socketPath, err)
	}
	return nil
}
