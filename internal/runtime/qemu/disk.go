package qemu

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strconv"
	"strings"
	rtm "yeast/internal/runtime"
)

const qemuImgBinary = "qemu-img"

var runCommand = runCommandContext

func PrepareDisk(ctx context.Context, plan rtm.MachinePlan) (rtm.DiskPlan, error) {
	disk := plan.Disk
	if strings.TrimSpace(disk.BaseImagePath) == "" {
		return rtm.DiskPlan{}, fmt.Errorf("base image path is required")
	}
	if strings.TrimSpace(disk.DiskPath) == "" {
		return rtm.DiskPlan{}, fmt.Errorf("disk path is required")
	}

	if plan.RuntimeDir != "" {
		if err := os.MkdirAll(plan.RuntimeDir, 0755); err != nil {
			return rtm.DiskPlan{}, fmt.Errorf("create runtime directory %s: %w", plan.RuntimeDir, err)
		}
	}

	diskDir := filepath.Dir(disk.DiskPath)
	if err := os.MkdirAll(diskDir, 0755); err != nil {
		return rtm.DiskPlan{}, fmt.Errorf("create disk directory %s: %w", diskDir, err)
	}

	if _, err := os.Stat(disk.DiskPath); err == nil {
		return disk, nil
	} else if !errorsIsNotExist(err) {
		return rtm.DiskPlan{}, fmt.Errorf("inspect disk path %s: %w", disk.DiskPath, err)
	}

	if err := runCommand(ctx, qemuImgBinary, buildCreateOverlayArgs(disk)...); err != nil {
		return rtm.DiskPlan{}, fmt.Errorf("prepare overlay disk %s: %w", disk.DiskPath, err)
	}

	return disk, nil
}

func buildCreateOverlayArgs(disk rtm.DiskPlan) []string {
	args := []string{
		"create",
		"-f", "qcow2",
		"-F", "qcow2",
		"-b", disk.BaseImagePath,
		"-o", "preallocation=metadata,extended_l2=on",
		disk.DiskPath,
	}
	if strings.TrimSpace(disk.Size) != "" {
		args = append(args, disk.Size)
	}
	return args
}

func buildDiskDriveArg(path string) string {
	aio := detectAIO()
	return fmt.Sprintf("file=%s,if=virtio,format=qcow2,cache=writeback,%s", filepath.Clean(path), aio)
}

func detectAIO() string {
	if goruntime.GOOS == "linux" && kernelVersionAtLeast(5, 1) {
		return "aio=io_uring"
	}
	return "aio=threads"
}

func kernelVersionAtLeast(major, minor int) bool {
	out, err := exec.Command("uname", "-r").Output()
	if err != nil {
		return false
	}
	parts := strings.SplitN(strings.TrimSpace(string(out)), ".", 3)
	if len(parts) < 2 {
		return false
	}
	maj, err := strconv.Atoi(parts[0])
	if err != nil {
		return false
	}
	min, err := strconv.Atoi(parts[1])
	if err != nil {
		return false
	}
	return maj > major || (maj == major && min >= minor)
}

func runCommandContext(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		trimmed := strings.TrimSpace(string(output))
		if trimmed == "" {
			return err
		}
		return fmt.Errorf("%w: %s", err, trimmed)
	}
	return nil
}

func errorsIsNotExist(err error) bool {
	return err != nil && os.IsNotExist(err)
}
