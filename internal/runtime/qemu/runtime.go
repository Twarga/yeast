package qemu

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
	rtm "yeast/internal/runtime"
)

type Runtime struct{}

const pidFileName = "qemu.pid"

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
	if err := proc.Release(); err != nil {
		return rtm.RuntimeInstance{}, fmt.Errorf("release qemu process handle: %w", err)
	}
	if err := writePIDFile(plan.RuntimeDir, proc.PID()); err != nil {
		_ = signalProcess(proc.PID(), syscall.SIGTERM)
		return rtm.RuntimeInstance{}, err
	}

	return rtm.RuntimeInstance{
		Name:       plan.Name,
		RuntimeDir: plan.RuntimeDir,
		LogPath:    plan.LogPath,
		PID:        proc.PID(),
		Networks:   plan.Networks,
		StartedAt:  time.Now().UTC(),
	}, nil
}

func (r *Runtime) Stop(ctx context.Context, instance rtm.RuntimeInstance, timeout time.Duration) error {
	if instance.PID <= 0 && instance.RuntimeDir != "" {
		pid, err := readPIDFile(instance.RuntimeDir)
		if err == nil {
			instance.PID = pid
		} else if os.IsNotExist(err) {
			return nil
		} else if !os.IsNotExist(err) {
			return err
		}
	}
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

	if err := signalProcess(instance.PID, syscall.SIGTERM); err != nil && !isNoSuchProcess(err) {
		return fmt.Errorf("send SIGTERM to process %d: %w", instance.PID, err)
	}
	if err := waitForProcessExit(ctx, instance.PID, timeout); err == nil {
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

	_ = os.Remove(pidFilePath(instance.RuntimeDir))
	return nil
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

func (r *Runtime) CleanOrphans(ctx context.Context, targets []rtm.CleanupTarget, timeout time.Duration) ([]rtm.CleanupResult, error) {
	processes, err := findQEMUProcesses(targets)
	if err != nil {
		return nil, err
	}

	results := make([]rtm.CleanupResult, 0, len(processes))
	for _, process := range processes {
		if err := stopPID(ctx, process.PID, timeout); err != nil {
			return results, err
		}
		if process.RuntimeDir != "" {
			_ = os.Remove(pidFilePath(process.RuntimeDir))
		}
		results = append(results, rtm.CleanupResult{Name: process.Name, PID: process.PID})
	}
	return results, nil
}

func writePIDFile(runtimeDir string, pid int) error {
	if runtimeDir == "" || pid <= 0 {
		return nil
	}
	path := pidFilePath(runtimeDir)
	if err := os.WriteFile(path, []byte(strconv.Itoa(pid)+"\n"), 0644); err != nil {
		return fmt.Errorf("write qemu pid file %s: %w", path, err)
	}
	return nil
}

func readPIDFile(runtimeDir string) (int, error) {
	path := pidFilePath(runtimeDir)
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil || pid <= 0 {
		return 0, fmt.Errorf("invalid qemu pid file %s", path)
	}
	return pid, nil
}

func pidFilePath(runtimeDir string) string {
	return filepath.Join(runtimeDir, pidFileName)
}

type qemuProcessMatch struct {
	Name       string
	RuntimeDir string
	PID        int
}

func findQEMUProcesses(targets []rtm.CleanupTarget) ([]qemuProcessMatch, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, fmt.Errorf("read /proc: %w", err)
	}

	seen := make(map[int]bool)
	var matches []qemuProcessMatch
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || seen[pid] {
			continue
		}
		cmdline, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "cmdline"))
		if err != nil || len(cmdline) == 0 {
			continue
		}
		args := strings.Split(strings.TrimRight(string(cmdline), "\x00"), "\x00")
		if !isQEMUSystemCommand(args) {
			continue
		}
		joined := strings.Join(args, " ")
		for _, target := range targets {
			if !targetMatchesCommand(target, joined) {
				continue
			}
			seen[pid] = true
			matches = append(matches, qemuProcessMatch{
				Name:       target.Name,
				RuntimeDir: target.RuntimeDir,
				PID:        pid,
			})
			break
		}
	}
	return matches, nil
}

func isQEMUSystemCommand(args []string) bool {
	if len(args) == 0 {
		return false
	}
	name := filepath.Base(args[0])
	return strings.HasPrefix(name, "qemu-system-")
}

func targetMatchesCommand(target rtm.CleanupTarget, cmdline string) bool {
	if strings.TrimSpace(target.RuntimeDir) != "" && strings.Contains(cmdline, filepath.Clean(target.RuntimeDir)) {
		return true
	}
	if target.SSHPort > 0 {
		host := strings.TrimSpace(target.SSHHost)
		if host == "" {
			host = "127.0.0.1"
		}
		return strings.Contains(cmdline, fmt.Sprintf("hostfwd=tcp:%s:%d-:22", host, target.SSHPort))
	}
	return false
}

func stopPID(ctx context.Context, pid int, timeout time.Duration) error {
	if pid <= 0 {
		return nil
	}
	if err := signalProcess(pid, syscall.SIGTERM); err != nil && !isNoSuchProcess(err) {
		return fmt.Errorf("send SIGTERM to orphan qemu process %d: %w", pid, err)
	}
	if err := waitForProcessExit(ctx, pid, timeout); err == nil {
		return nil
	} else if err != context.DeadlineExceeded {
		return fmt.Errorf("wait for orphan qemu process %d after SIGTERM: %w", pid, err)
	}
	if err := signalProcess(pid, syscall.SIGKILL); err != nil && !isNoSuchProcess(err) {
		return fmt.Errorf("send SIGKILL to orphan qemu process %d: %w", pid, err)
	}
	if err := waitForProcessExit(ctx, pid, 5*time.Second); err != nil && err != context.DeadlineExceeded {
		return fmt.Errorf("wait for orphan qemu process %d after SIGKILL: %w", pid, err)
	}
	return nil
}
