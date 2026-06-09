package qemu

import (
	"fmt"
	"hash/fnv"
	"net/netip"
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
	if strings.TrimSpace(plan.Networks.Management.InterfaceName) == "" {
		return nil, fmt.Errorf("management network interface name is required")
	}
	if strings.TrimSpace(plan.Networks.Management.MACAddress) == "" {
		return nil, fmt.Errorf("management network mac address is required")
	}
	if plan.Networks.Lab != nil {
		if strings.TrimSpace(plan.Networks.Lab.Name) == "" {
			return nil, fmt.Errorf("lab network name is required")
		}
		if strings.TrimSpace(plan.Networks.Lab.InterfaceName) == "" {
			return nil, fmt.Errorf("lab network interface name is required")
		}
		if strings.TrimSpace(plan.Networks.Lab.MACAddress) == "" {
			return nil, fmt.Errorf("lab network mac address is required")
		}
		if !plan.Networks.Lab.CIDR.IsValid() {
			return nil, fmt.Errorf("lab network cidr is required")
		}
		if !plan.Networks.Lab.IPv4.IsValid() || !plan.Networks.Lab.IPv4.Is4() {
			return nil, fmt.Errorf("lab network ipv4 must be a valid IPv4 address")
		}
		if !plan.Networks.Lab.CIDR.Contains(plan.Networks.Lab.IPv4) {
			return nil, fmt.Errorf("lab network ipv4 %s is outside cidr %s", plan.Networks.Lab.IPv4, plan.Networks.Lab.CIDR)
		}
	}

	args := []string{
		"-enable-kvm",
		"-name", plan.Name,
		"-m", fmt.Sprintf("%d", plan.MemoryMiB),
		"-smp", fmt.Sprintf("%d", plan.CPUs),
		"-drive", buildDiskDriveArg(plan.Disk.DiskPath),
		"-drive", buildSeedDriveArg(plan.SeedImagePath),
		"-netdev", buildManagementNetdevArg(plan.Networks.Management.SSHHost, plan.Networks.Management.SSHPort),
		"-device", buildManagementDeviceArg(plan.Networks.Management),
		"-qmp", fmt.Sprintf("unix:%s,server,nowait", qmpSocketPath(plan.RuntimeDir)),
	}
	if plan.Networks.Lab != nil {
		args = append(args,
			"-netdev", buildLabNetdevArg(plan.RuntimeDir, *plan.Networks.Lab),
			"-device", buildLabDeviceArg(*plan.Networks.Lab),
		)
	}
	args = append(args, "-nographic")

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

func buildManagementDeviceArg(management runtime.ManagementNetworkPlan) string {
	return fmt.Sprintf("virtio-net-pci,netdev=mgmt0,mac=%s", management.MACAddress)
}

func buildLabNetdevArg(runtimeDir string, lab runtime.LabNetworkPlan) string {
	group, port := deriveLabMulticast(filepath.Clean(filepath.Dir(runtimeDir)), lab.Name)
	return fmt.Sprintf("socket,id=lab0,mcast=%s:%d,localaddr=127.0.0.1", group, port)
}

func buildLabDeviceArg(lab runtime.LabNetworkPlan) string {
	return fmt.Sprintf("virtio-net-pci,netdev=lab0,mac=%s", lab.MACAddress)
}

func deriveLabMulticast(projectScope string, networkName string) (string, int) {
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(projectScope))
	_, _ = hash.Write([]byte("|"))
	_, _ = hash.Write([]byte(networkName))
	sum := hash.Sum32()

	third := byte((sum >> 8) & 0xff)
	fourth := byte(sum & 0xff)
	if third == 0 {
		third = 1
	}
	if fourth == 0 {
		fourth = 1
	}
	group := netip.AddrFrom4([4]byte{239, 192, third, fourth})
	port := 10000 + int(sum%40000)
	return group.String(), port
}
