package qemu

import (
	"fmt"
	"path/filepath"
	"strings"
	"yeast/internal/runtime"
)

const qemuSystemBinary = "qemu-system-x86_64"

func BuildCommandArgs(plan runtime.MachinePlan) ([]string, error) {
	if strings.TrimSpace(plan.Name) == "" {
		return nil, fmt.Errorf("machine name is required")
	}
	if plan.MemoryMiB <= 0 {
		return nil, fmt.Errorf("memory must be greater than zero")
	}
	if plan.CPUs <= 0 {
		return nil, fmt.Errorf("cpus must be greater than zero")
	}
	if strings.TrimSpace(plan.Disk.DiskPath) == "" {
		return nil, fmt.Errorf("disk path is required")
	}
	if strings.TrimSpace(plan.SeedImagePath) == "" {
		return nil, fmt.Errorf("seed image path is required")
	}
	if plan.Networks.Management.SSHPort <= 0 {
		return nil, fmt.Errorf("management ssh port must be greater than zero")
	}
	if strings.TrimSpace(plan.Networks.Management.SSHHost) == "" {
		return nil, fmt.Errorf("management ssh host is required")
	}

	args := []string{
		"-enable-kvm",
		"-name", plan.Name,
		"-m", fmt.Sprintf("%d", plan.MemoryMiB),
		"-smp", fmt.Sprintf("%d", plan.CPUs),
		"-drive", buildDiskDriveArg(plan.Disk.DiskPath),
		"-drive", buildSeedDriveArg(plan.SeedImagePath),
		"-netdev", buildManagementNetdevArg(plan.Networks.Management.SSHHost, plan.Networks.Management.SSHPort),
		"-device", "virtio-net-pci,netdev=mgmt0",
		"-nographic",
	}

	return args, nil
}

func buildCommand(plan runtime.MachinePlan) (string, []string, error) {
	args, err := BuildCommandArgs(plan)
	if err != nil {
		return "", nil, err
	}
	return qemuSystemBinary, args, nil
}

func buildDiskDriveArg(path string) string {
	return fmt.Sprintf("file=%s,if=virtio,format=qcow2", filepath.Clean(path))
}

func buildSeedDriveArg(path string) string {
	return fmt.Sprintf("file=%s,if=virtio,media=cdrom,readonly=on", filepath.Clean(path))
}

func buildManagementNetdevArg(host string, port int) string {
	return fmt.Sprintf("user,id=mgmt0,hostfwd=tcp:%s:%d-:22", host, port)
}
