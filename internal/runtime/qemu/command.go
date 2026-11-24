package qemu

import (
	"fmt"
	"github.com/Twarga/yeast/internal/runtime"
	"hash/fnv"
	"net/netip"
	"path/filepath"
	"strings"
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
	if strings.TrimSpace(plan.SeedImagePath) == "" && strings.TrimSpace(plan.BootISOPath) == "" {
		return nil, fmt.Errorf("seed image path is required")
	}
	if strings.TrimSpace(plan.SeedImagePath) != "" && strings.TrimSpace(plan.BootISOPath) != "" {
		return nil, fmt.Errorf("seed image and boot ISO are mutually exclusive")
	}
	switch plan.Firmware {
	case "", "bios", "uefi":
	default:
		return nil, fmt.Errorf("unsupported firmware %q (expected bios or uefi)", plan.Firmware)
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
		"-machine", "q35,accel=kvm",
		"-cpu", "host",
		"-enable-kvm",
		"-name", plan.Name,
		"-m", fmt.Sprintf("%d", plan.MemoryMiB),
		"-smp", fmt.Sprintf("%d", plan.CPUs),
	}
	if plan.Firmware == "uefi" {
		code, _, err := discoverFirmwareFn()
		if err != nil {
			return nil, err
		}
		args = append(args,
			"-drive", fmt.Sprintf("if=pflash,format=raw,readonly=on,file=%s", filepath.Clean(code)),
			"-drive", fmt.Sprintf("if=pflash,format=raw,file=%s", filepath.Join(plan.RuntimeDir, ovmfVarsName)),
		)
	}
	args = append(args,
		"-drive", buildDiskDriveArg(plan.Disk.DiskPath),
	)
	if plan.BootISOPath != "" {
		args = append(args,
			"-drive", buildBootISODriveArg(plan.BootISOPath),
			"-boot", "order=d",
		)
	} else {
		args = append(args,
			"-drive", buildSeedDriveArg(plan.SeedImagePath),
		)
	}
	args = append(args,
		"-device", "virtio-rng-pci",
		"-netdev", buildManagementNetdevArgFull(plan.Networks.Management.SSHHost, plan.Networks.Management.SSHPort, plan.Networks.Management.PortForwards, plan.Egress),
		"-device", buildManagementDeviceArg(plan.Networks.Management),
		"-qmp", fmt.Sprintf("unix:%s,server,nowait", qmpSocketPath(plan.RuntimeDir)),
	)
	args = append(args, sandboxArgs(plan)...)
	if plan.IncomingDefer {
		args = append(args, "-incoming", "defer")
	}
	if plan.Networks.Lab != nil {
		args = append(args,
			"-netdev", buildLabNetdevArg(plan.RuntimeDir, *plan.Networks.Lab),
			"-device", buildLabDeviceArg(*plan.Networks.Lab),
		)
	}
	if plan.Display != nil {
		displayArgs, err := buildDisplayArgs(plan)
		if err != nil {
			return nil, err
		}
		args = append(args, displayArgs...)
	} else {
		args = append(args, "-nographic")
	}
	args = append(args, buildMountArgs(plan)...)

	return args, nil
}

// buildDisplayArgs renders -vnc + GPU + input + serial args for a graphical
// machine. plan.Display must be non-nil (checked by caller).
func buildDisplayArgs(plan runtime.MachinePlan) ([]string, error) {
	d := plan.Display
	if strings.TrimSpace(d.Host) == "" {
		return nil, fmt.Errorf("display host is required")
	}
	if d.Port < 5900 || d.Port > 65535 {
		return nil, fmt.Errorf("display port must be in 5900..65535 (got %d)", d.Port)
	}

	args := []string{
		"-vnc", fmt.Sprintf("%s:%d", d.Host, d.Port-5900),
		"-device", "qemu-xhci",
		"-device", "usb-tablet",
		"-device", "usb-kbd",
		"-serial", fmt.Sprintf("file:%s", filepath.Join(plan.RuntimeDir, "console.log")),
	}

	res := ""
	if d.Width > 0 && d.Height > 0 {
		res = fmt.Sprintf(",xres=%d,yres=%d", d.Width, d.Height)
	}
	switch d.GPU {
	case "virtio-gpu":
		args = append(args, "-device", "virtio-gpu-pci"+res)
	case "virgl":
		renderNode := strings.TrimSpace(d.RenderNode)
		if renderNode == "" {
			renderNode = "/dev/dri/renderD128"
		}
		args = append(args,
			"-device", "virtio-vga-gl"+res,
			"-display", "egl-headless,rendernode="+renderNode,
		)
	case "", "virtio-vga":
		args = append(args, "-device", "virtio-vga"+res)
	default:
		return nil, fmt.Errorf("unsupported display gpu %q (expected virtio-vga, virtio-gpu, or virgl)", d.GPU)
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

func buildSeedDriveArg(path string) string {
	return fmt.Sprintf("file=%s,if=virtio,media=cdrom,readonly=on", filepath.Clean(path))
}

func buildBootISODriveArg(path string) string {
	return fmt.Sprintf("file=%s,if=ide,media=cdrom,readonly=on", filepath.Clean(path))
}

func buildManagementNetdevArg(host string, port int) string {
	return buildManagementNetdevArgWithForwards(host, port, nil)
}

func buildManagementNetdevArgWithForwards(host string, port int, forwards []runtime.PortForwardPlan) string {
	return buildManagementNetdevArgFull(host, port, forwards, nil)
}

func buildManagementNetdevArgFull(host string, port int, forwards []runtime.PortForwardPlan, egress *runtime.EgressPlan) string {
	parts := []string{
		"user",
		"id=mgmt0",
		fmt.Sprintf("hostfwd=tcp:%s:%d-:22", host, port),
	}
	for _, forward := range forwards {
		parts = append(parts, fmt.Sprintf("hostfwd=%s:%s:%d-:%d", forward.Protocol, forward.Host, forward.HostPort, forward.GuestPort))
	}
	if egress != nil {
		switch egress.Mode {
		case "offline":
			// restrict=on isolates the guest: no internet, no host — SSH
			// hostfwd and lab networking keep working.
			parts = append(parts, "restrict=on")
		case "allowlist":
			parts = append(parts, "restrict=on",
				fmt.Sprintf("guestfwd=tcp:10.0.2.100:3128-tcp:127.0.0.1:%d", egress.ProxyPort))
		}
	}
	return strings.Join(parts, ",")
}

// sandboxArgs returns the -sandbox hardening flags. Full restrictions apply
// by default; a virgl display needs host-GPU ioctls, so resourcecontrol is
// relaxed only then.
func sandboxArgs(plan runtime.MachinePlan) []string {
	restrictions := "on,obsolete=deny,elevateprivileges=deny,spawn=deny,resourcecontrol=deny"
	if plan.Display != nil && plan.Display.GPU == "virgl" {
		restrictions = "on,obsolete=deny,elevateprivileges=deny,spawn=deny"
	}
	return []string{"-sandbox", restrictions}
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
