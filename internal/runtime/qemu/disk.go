package qemu

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"yeast/internal/runtime"
)

const qemuImgBinary = "qemu-img"

var runCommand = runCommandContext

func PrepareDisk(ctx context.Context, plan runtime.MachinePlan) (runtime.DiskPlan, error) {
	disk := plan.Disk
	if strings.TrimSpace(disk.BaseImagePath) == "" {
		return runtime.DiskPlan{}, fmt.Errorf("base image path is required")
	}
	if strings.TrimSpace(disk.DiskPath) == "" {
		return runtime.DiskPlan{}, fmt.Errorf("disk path is required")
	}

	if plan.RuntimeDir != "" {
		if err := os.MkdirAll(plan.RuntimeDir, 0755); err != nil {
			return runtime.DiskPlan{}, fmt.Errorf("create runtime directory %s: %w", plan.RuntimeDir, err)
		}
	}

	diskDir := filepath.Dir(disk.DiskPath)
	if err := os.MkdirAll(diskDir, 0755); err != nil {
		return runtime.DiskPlan{}, fmt.Errorf("create disk directory %s: %w", diskDir, err)
	}

	if _, err := os.Stat(disk.DiskPath); err == nil {
		return disk, nil
	} else if !errorsIsNotExist(err) {
		return runtime.DiskPlan{}, fmt.Errorf("inspect disk path %s: %w", disk.DiskPath, err)
	}

	if err := runCommand(ctx, qemuImgBinary, buildCreateOverlayArgs(disk)...); err != nil {
		return runtime.DiskPlan{}, fmt.Errorf("prepare overlay disk %s: %w", disk.DiskPath, err)
	}

	return disk, nil
}

func buildCreateOverlayArgs(disk runtime.DiskPlan) []string {
	args := []string{
		"create",
		"-f", "qcow2",
		"-F", "qcow2",
		"-b", disk.BaseImagePath,
		disk.DiskPath,
	}
	if strings.TrimSpace(disk.Size) != "" {
		args = append(args, disk.Size)
	}
	return args
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
