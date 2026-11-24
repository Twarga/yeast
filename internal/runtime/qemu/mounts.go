package qemu

import (
	"context"
	"fmt"
	rtm "github.com/Twarga/yeast/internal/runtime"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// buildMountArgs renders the QEMU args for shared host→guest mounts.
// virtiofs mounts additionally require shared memory (memfd + numa memdev)
// which replaces QEMU's default anonymous RAM for those machines.
func buildMountArgs(plan rtm.MachinePlan) []string {
	if len(plan.Mounts) == 0 {
		return nil
	}
	var args []string
	for _, m := range plan.Mounts {
		if m.Driver == "virtiofs" {
			args = append(args,
				"-object", fmt.Sprintf("memory-backend-memfd,id=mem,size=%dM,share=on", plan.MemoryMiB),
				"-numa", "node,memdev=mem",
			)
			break
		}
	}
	for i, m := range plan.Mounts {
		switch m.Driver {
		case "virtiofs":
			args = append(args,
				"-chardev", fmt.Sprintf("socket,id=vfs%d,path=%s", i, virtiofsSocketPath(plan.RuntimeDir, i)),
				"-device", fmt.Sprintf("vhost-user-fs-pci,chardev=vfs%d,tag=%s", i, m.Tag),
			)
		case "9p":
			fsdev := fmt.Sprintf("local,id=fs%d,path=%s,security_model=mapped-xattr", i, filepath.Clean(m.Source))
			if m.ReadOnly {
				fsdev += ",readonly=on"
			}
			args = append(args,
				"-fsdev", fsdev,
				"-device", fmt.Sprintf("virtio-9p-pci,fsdev=fs%d,mount_tag=%s", i, m.Tag),
			)
		}
	}
	return args
}

func virtiofsSocketPath(runtimeDir string, idx int) string {
	return filepath.Join(runtimeDir, fmt.Sprintf("vfs-%d.sock", idx))
}

func virtiofsPIDPath(runtimeDir string, idx int) string {
	return filepath.Join(runtimeDir, fmt.Sprintf("vfs-%d.pid", idx))
}

// spawnMountSidecars starts one rootless virtiofsd per virtiofs mount and
// records each pid in <RuntimeDir>/vfs-<idx>.pid so later yeast invocations
// (down/destroy on a fresh process) can still reap them.
func spawnMountSidecars(ctx context.Context, plan rtm.MachinePlan) error {
	for i, m := range plan.Mounts {
		if m.Driver != "virtiofs" {
			continue
		}
		if strings.TrimSpace(m.DaemonPath) == "" {
			return fmt.Errorf("mount %d (%s): virtiofs daemon path is empty", i, m.Tag)
		}
		sockPath := virtiofsSocketPath(plan.RuntimeDir, i)
		if err := os.Remove(sockPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove stale virtiofs socket %s: %w", sockPath, err)
		}
		args := []string{
			"--socket-path=" + sockPath,
			"--shared-dir=" + filepath.Clean(m.Source),
			"--sandbox=none",
		}
		if m.ReadOnly {
			args = append(args, "--readonly")
		}
		// No CommandContext: the sidecar must outlive this CLI invocation,
		// same as the QEMU process itself.
		cmd := exec.Command(m.DaemonPath, args...)
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		if err := cmd.Start(); err != nil {
			stopMountSidecars(plan.RuntimeDir)
			return fmt.Errorf("start virtiofsd for %s: %w", m.Source, err)
		}
		pid := cmd.Process.Pid
		if err := os.WriteFile(virtiofsPIDPath(plan.RuntimeDir, i), []byte(strconv.Itoa(pid)), 0o644); err != nil {
			_ = cmd.Process.Kill()
			stopMountSidecars(plan.RuntimeDir)
			return fmt.Errorf("record virtiofsd pid: %w", err)
		}
		if err := cmd.Process.Release(); err != nil {
			stopMountSidecars(plan.RuntimeDir)
			return fmt.Errorf("release virtiofsd handle: %w", err)
		}
		if err := waitForSocketFile(sockPath, 3*time.Second); err != nil {
			stopMountSidecars(plan.RuntimeDir)
			return fmt.Errorf("virtiofsd socket for %s did not appear: %w", m.Source, err)
		}
	}
	return nil
}

func waitForSocketFile(path string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		if _, err := os.Stat(path); err == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for %s", path)
		}
		time.Sleep(25 * time.Millisecond)
	}
}

// stopMountSidecars SIGTERMs every recorded virtiofsd pidfile in runtimeDir
// and removes the pid/socket leftovers. Best-effort; never fails the caller.
func stopMountSidecars(runtimeDir string) {
	if runtimeDir == "" {
		return
	}
	pidFiles, err := filepath.Glob(filepath.Join(runtimeDir, "vfs-*.pid"))
	if err != nil {
		return
	}
	for _, pidFile := range pidFiles {
		data, err := os.ReadFile(pidFile)
		if err == nil {
			if pid, convErr := strconv.Atoi(strings.TrimSpace(string(data))); convErr == nil && pid > 0 {
				_ = signalProcessFn(pid, syscall.SIGTERM)
			}
		}
		_ = os.Remove(pidFile)
	}
	socks, err := filepath.Glob(filepath.Join(runtimeDir, "vfs-*.sock"))
	if err != nil {
		return
	}
	for _, sock := range socks {
		_ = os.Remove(sock)
	}
}
