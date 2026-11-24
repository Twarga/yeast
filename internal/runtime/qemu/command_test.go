package qemu

import (
	"fmt"
	"github.com/Twarga/yeast/internal/runtime"
	"net/netip"
	"reflect"
	"strings"
	"testing"
)

func TestBuildCommandArgsIncludesExpectedRuntimeFlags(t *testing.T) {
	t.Parallel()

	plan := runtime.MachinePlan{
		Name:          "web",
		RuntimeDir:    "/runtime/web",
		LogPath:       "/runtime/web/vm.log",
		MemoryMiB:     2048,
		CPUs:          2,
		SeedImagePath: "/runtime/web/seed.iso",
		Disk: runtime.DiskPlan{
			DiskPath: "/runtime/web/disk.qcow2",
		},
		Networks: runtime.NetworkPlan{
			Management: runtime.ManagementNetworkPlan{
				SSHHost:       "127.0.0.1",
				SSHPort:       2222,
				InterfaceName: "yeastmgmt0",
				MACAddress:    "52:54:00:11:22:33",
			},
		},
	}

	got, err := BuildCommandArgs(plan)
	if err != nil {
		t.Fatalf("BuildCommandArgs returned error: %v", err)
	}

	want := []string{
		"-machine", "q35,accel=kvm",
		"-cpu", "host",
		"-enable-kvm",
		"-name", "web",
		"-m", "2048",
		"-smp", "2",
		"-drive", buildDiskDriveArg("/runtime/web/disk.qcow2"),
		"-drive", "file=/runtime/web/seed.iso,if=virtio,media=cdrom,readonly=on",
		"-device", "virtio-rng-pci",
		"-netdev", "user,id=mgmt0,hostfwd=tcp:127.0.0.1:2222-:22",
		"-device", "virtio-net-pci,netdev=mgmt0,mac=52:54:00:11:22:33",
		"-qmp", "unix:/runtime/web/qmp.sock,server,nowait",
		"-sandbox", "on,obsolete=deny,elevateprivileges=deny,spawn=deny,resourcecontrol=deny",
		"-nographic",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected args:\n got: %#v\nwant: %#v", got, want)
	}
}

func TestBuildCommandArgsIncludesLabNetworkFlags(t *testing.T) {
	t.Parallel()

	plan := runtime.MachinePlan{
		Name:          "target",
		RuntimeDir:    "/runtime/proj/instances/target",
		LogPath:       "/runtime/proj/instances/target/vm.log",
		MemoryMiB:     1024,
		CPUs:          1,
		SeedImagePath: "/runtime/proj/instances/target/seed.iso",
		Disk: runtime.DiskPlan{
			DiskPath: "/runtime/proj/instances/target/disk.qcow2",
		},
		Networks: runtime.NetworkPlan{
			Management: runtime.ManagementNetworkPlan{
				SSHHost:       "127.0.0.1",
				SSHPort:       2222,
				InterfaceName: "yeastmgmt0",
				MACAddress:    "52:54:00:11:22:33",
			},
			Lab: &runtime.LabNetworkPlan{
				Name:          "lab",
				CIDR:          netip.MustParsePrefix("10.10.10.0/24"),
				IPv4:          netip.MustParseAddr("10.10.10.20"),
				InterfaceName: "yeastlab0",
				MACAddress:    "52:54:00:aa:bb:cc",
			},
		},
	}

	got, err := BuildCommandArgs(plan)
	if err != nil {
		t.Fatalf("BuildCommandArgs returned error: %v", err)
	}

	wantLabNetdev := buildLabNetdevArg(plan.RuntimeDir, *plan.Networks.Lab)
	want := []string{
		"-machine", "q35,accel=kvm",
		"-cpu", "host",
		"-enable-kvm",
		"-name", "target",
		"-m", "1024",
		"-smp", "1",
		"-drive", buildDiskDriveArg("/runtime/proj/instances/target/disk.qcow2"),
		"-drive", "file=/runtime/proj/instances/target/seed.iso,if=virtio,media=cdrom,readonly=on",
		"-device", "virtio-rng-pci",
		"-netdev", "user,id=mgmt0,hostfwd=tcp:127.0.0.1:2222-:22",
		"-device", "virtio-net-pci,netdev=mgmt0,mac=52:54:00:11:22:33",
		"-qmp", "unix:/runtime/proj/instances/target/qmp.sock,server,nowait",
		"-sandbox", "on,obsolete=deny,elevateprivileges=deny,spawn=deny,resourcecontrol=deny",
		"-netdev", wantLabNetdev,
		"-device", "virtio-net-pci,netdev=lab0,mac=52:54:00:aa:bb:cc",
		"-nographic",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected args:\n got: %#v\nwant: %#v", got, want)
	}
}

func TestBuildCommandArgsIncludesServicePortForwards(t *testing.T) {
	t.Parallel()

	plan := runtime.MachinePlan{
		Name:          "web",
		RuntimeDir:    "/runtime/web",
		LogPath:       "/runtime/web/vm.log",
		MemoryMiB:     2048,
		CPUs:          2,
		SeedImagePath: "/runtime/web/seed.iso",
		Disk: runtime.DiskPlan{
			DiskPath: "/runtime/web/disk.qcow2",
		},
		Networks: runtime.NetworkPlan{
			Management: runtime.ManagementNetworkPlan{
				SSHHost:       "127.0.0.1",
				SSHPort:       2222,
				InterfaceName: "yeastmgmt0",
				MACAddress:    "52:54:00:11:22:33",
				PortForwards: []runtime.PortForwardPlan{
					{Host: "127.0.0.1", HostPort: 8080, GuestPort: 80, Protocol: "tcp"},
					{Host: "0.0.0.0", HostPort: 3000, GuestPort: 3000, Protocol: "tcp"},
				},
			},
		},
	}

	got, err := BuildCommandArgs(plan)
	if err != nil {
		t.Fatalf("BuildCommandArgs returned error: %v", err)
	}

	found := false
	for i := 0; i < len(got)-1; i++ {
		if got[i] == "-netdev" && got[i+1] == "user,id=mgmt0,hostfwd=tcp:127.0.0.1:2222-:22,hostfwd=tcp:127.0.0.1:8080-:80,hostfwd=tcp:0.0.0.0:3000-:3000" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected service forwards in netdev args, got %#v", got)
	}
}

func TestBuildCommandReturnsBinaryAndArgs(t *testing.T) {
	t.Parallel()

	plan := runtime.MachinePlan{
		Name:          "db",
		MemoryMiB:     1024,
		CPUs:          1,
		SeedImagePath: "/runtime/db/seed.iso",
		Disk: runtime.DiskPlan{
			DiskPath: "/runtime/db/disk.qcow2",
		},
		Networks: runtime.NetworkPlan{
			Management: runtime.ManagementNetworkPlan{
				SSHHost:       "127.0.0.1",
				SSHPort:       2201,
				InterfaceName: "yeastmgmt0",
				MACAddress:    "52:54:00:11:22:33",
			},
		},
	}

	binary, args, err := buildCommand(plan)
	if err != nil {
		t.Fatalf("buildCommand returned error: %v", err)
	}
	if binary != qemuSystemBinary {
		t.Fatalf("unexpected binary: got %q want %q", binary, qemuSystemBinary)
	}
	if len(args) == 0 {
		t.Fatal("expected non-empty args")
	}
}

func TestBuildCommandArgsRejectsMissingFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		plan runtime.MachinePlan
	}{
		{
			name: "missing name",
			plan: runtime.MachinePlan{
				MemoryMiB: 1024,
				CPUs:      1,
				Disk: runtime.DiskPlan{
					DiskPath: "/runtime/web/disk.qcow2",
				},
				SeedImagePath: "/runtime/web/seed.iso",
				Networks: runtime.NetworkPlan{
					Management: runtime.ManagementNetworkPlan{
						SSHHost:       "127.0.0.1",
						SSHPort:       2222,
						InterfaceName: "yeastmgmt0",
						MACAddress:    "52:54:00:11:22:33",
					},
				},
			},
		},
		{
			name: "missing disk path",
			plan: runtime.MachinePlan{
				Name:          "web",
				MemoryMiB:     1024,
				CPUs:          1,
				SeedImagePath: "/runtime/web/seed.iso",
				Networks: runtime.NetworkPlan{
					Management: runtime.ManagementNetworkPlan{
						SSHHost:       "127.0.0.1",
						SSHPort:       2222,
						InterfaceName: "yeastmgmt0",
						MACAddress:    "52:54:00:11:22:33",
					},
				},
			},
		},
		{
			name: "missing seed image",
			plan: runtime.MachinePlan{
				Name:      "web",
				MemoryMiB: 1024,
				CPUs:      1,
				Disk: runtime.DiskPlan{
					DiskPath: "/runtime/web/disk.qcow2",
				},
				Networks: runtime.NetworkPlan{
					Management: runtime.ManagementNetworkPlan{
						SSHHost:       "127.0.0.1",
						SSHPort:       2222,
						InterfaceName: "yeastmgmt0",
						MACAddress:    "52:54:00:11:22:33",
					},
				},
			},
		},
		{
			name: "missing management port",
			plan: runtime.MachinePlan{
				Name:          "web",
				MemoryMiB:     1024,
				CPUs:          1,
				SeedImagePath: "/runtime/web/seed.iso",
				Disk: runtime.DiskPlan{
					DiskPath: "/runtime/web/disk.qcow2",
				},
			},
		},
		{
			name: "lab ipv4 outside cidr",
			plan: runtime.MachinePlan{
				Name:          "web",
				MemoryMiB:     1024,
				CPUs:          1,
				SeedImagePath: "/runtime/web/seed.iso",
				Disk: runtime.DiskPlan{
					DiskPath: "/runtime/web/disk.qcow2",
				},
				Networks: runtime.NetworkPlan{
					Management: runtime.ManagementNetworkPlan{
						SSHHost:       "127.0.0.1",
						SSHPort:       2222,
						InterfaceName: "yeastmgmt0",
						MACAddress:    "52:54:00:11:22:33",
					},
					Lab: &runtime.LabNetworkPlan{
						Name:          "lab",
						CIDR:          netip.MustParsePrefix("10.10.10.0/24"),
						IPv4:          netip.MustParseAddr("10.20.20.20"),
						InterfaceName: "yeastlab0",
						MACAddress:    "52:54:00:aa:bb:cc",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, err := BuildCommandArgs(tt.plan); err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestDeriveLabMulticastIsStablePerProjectScopeAndNetwork(t *testing.T) {
	t.Parallel()

	groupA, portA := deriveLabMulticast("/runtime/proj/instances", "lab")
	groupB, portB := deriveLabMulticast("/runtime/proj/instances", "lab")
	groupOther, portOther := deriveLabMulticast("/runtime/proj-other/instances", "lab")

	if groupA != groupB || portA != portB {
		t.Fatalf("expected stable multicast derivation, got %s:%d and %s:%d", groupA, portA, groupB, portB)
	}
	if groupA == groupOther && portA == portOther {
		t.Fatalf("expected different project scopes to derive different multicast endpoints, got same %s:%d", groupA, portA)
	}
}

func basePlan() runtime.MachinePlan {
	return runtime.MachinePlan{
		Name:          "web",
		RuntimeDir:    "/runtime/web",
		LogPath:       "/runtime/web/vm.log",
		MemoryMiB:     2048,
		CPUs:          2,
		SeedImagePath: "/runtime/web/seed.iso",
		Disk:          runtime.DiskPlan{DiskPath: "/runtime/web/disk.qcow2"},
		Networks: runtime.NetworkPlan{
			Management: runtime.ManagementNetworkPlan{
				SSHHost: "127.0.0.1", SSHPort: 2222,
				InterfaceName: "yeastmgmt0", MACAddress: "52:54:00:11:22:33",
			},
		},
	}
}

func tailArgs(args []string, n int) []string {
	if len(args) < n {
		return args
	}
	return args[len(args)-n:]
}

func TestBuildCommandArgsVNCDisplay(t *testing.T) {
	plan := basePlan()
	plan.Display = &runtime.DisplayPlan{
		Type: "vnc", Host: "127.0.0.1", Port: 5901, GPU: "virtio-vga",
		Width: 1280, Height: 800,
	}
	got, err := BuildCommandArgs(plan)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range got {
		if a == "-nographic" {
			t.Fatal("display mode must not emit -nographic")
		}
	}
	want := []string{
		"-vnc", "127.0.0.1:1",
		"-device", "qemu-xhci",
		"-device", "usb-tablet",
		"-device", "usb-kbd",
		"-serial", "file:/runtime/web/console.log",
		"-device", "virtio-vga,xres=1280,yres=800",
	}
	if !reflect.DeepEqual(tailArgs(got, len(want)), want) {
		t.Fatalf("display args:\n got: %#v\nwant: %#v", tailArgs(got, len(want)), want)
	}
}

func TestBuildCommandArgsVNCNoResolution(t *testing.T) {
	plan := basePlan()
	plan.Display = &runtime.DisplayPlan{Type: "vnc", Host: "127.0.0.1", Port: 5910, GPU: "virtio-gpu"}
	got, err := BuildCommandArgs(plan)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"-vnc", "127.0.0.1:10",
		"-device", "qemu-xhci",
		"-device", "usb-tablet",
		"-device", "usb-kbd",
		"-serial", "file:/runtime/web/console.log",
		"-device", "virtio-gpu-pci",
	}
	if !reflect.DeepEqual(tailArgs(got, len(want)), want) {
		t.Fatalf("display args:\n got: %#v\nwant: %#v", tailArgs(got, len(want)), want)
	}
}

func TestBuildCommandArgsVirgl(t *testing.T) {
	plan := basePlan()
	plan.Display = &runtime.DisplayPlan{
		Type: "vnc", Host: "127.0.0.1", Port: 5901, GPU: "virgl",
		RenderNode: "/dev/dri/renderD129", Width: 1920, Height: 1080,
	}
	got, err := BuildCommandArgs(plan)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"-vnc", "127.0.0.1:1",
		"-device", "qemu-xhci",
		"-device", "usb-tablet",
		"-device", "usb-kbd",
		"-serial", "file:/runtime/web/console.log",
		"-device", "virtio-vga-gl,xres=1920,yres=1080",
		"-display", "egl-headless,rendernode=/dev/dri/renderD129",
	}
	if !reflect.DeepEqual(tailArgs(got, len(want)), want) {
		t.Fatalf("virgl args:\n got: %#v\nwant: %#v", tailArgs(got, len(want)), want)
	}
}

func TestBuildCommandArgsDisplayRejects(t *testing.T) {
	plan := basePlan()
	plan.Display = &runtime.DisplayPlan{Type: "vnc", Port: 5800, GPU: "virtio-vga"}
	if _, err := BuildCommandArgs(plan); err == nil {
		t.Fatal("expected port range error")
	}
	plan.Display = &runtime.DisplayPlan{Type: "vnc", Host: "127.0.0.1", Port: 5901, GPU: "nvidia"}
	if _, err := BuildCommandArgs(plan); err == nil {
		t.Fatal("expected gpu error")
	}
}

func TestBuildCommandArgsUEFI(t *testing.T) {
	defer func(f func() (string, string, error)) { discoverFirmwareFn = f }(discoverFirmwareFn)
	discoverFirmwareFn = func() (string, string, error) {
		return "/usr/share/OVMF/OVMF_CODE.fd", "/usr/share/OVMF/OVMF_VARS.fd", nil
	}
	plan := basePlan()
	plan.Firmware = "uefi"
	got, err := BuildCommandArgs(plan)
	if err != nil {
		t.Fatal(err)
	}
	// pflash drives must precede the virtio disk drive
	var pflashIdx, diskIdx int = -1, -1
	for i, a := range got {
		if a == "if=pflash,format=raw,readonly=on,file=/usr/share/OVMF/OVMF_CODE.fd" {
			pflashIdx = i
		}
		if a == buildDiskDriveArg("/runtime/web/disk.qcow2") {
			diskIdx = i
		}
	}
	if pflashIdx == -1 || diskIdx == -1 || pflashIdx > diskIdx {
		t.Fatalf("pflash ordering wrong:\n%#v", got)
	}
	vars := "if=pflash,format=raw,file=/runtime/web/ovmf_vars.fd"
	found := false
	for _, a := range got {
		if a == vars {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing vars drive:\n%#v", got)
	}
}

func TestBuildCommandArgsUEFIMissing(t *testing.T) {
	defer func(f func() (string, string, error)) { discoverFirmwareFn = f }(discoverFirmwareFn)
	discoverFirmwareFn = func() (string, string, error) {
		return "", "", fmt.Errorf("no OVMF UEFI firmware found")
	}
	plan := basePlan()
	plan.Firmware = "uefi"
	if _, err := BuildCommandArgs(plan); err == nil {
		t.Fatal("expected OVMF-missing error")
	}
}

func TestBuildCommandArgsInvalidFirmware(t *testing.T) {
	plan := basePlan()
	plan.Firmware = "coreboot"
	if _, err := BuildCommandArgs(plan); err == nil {
		t.Fatal("expected firmware error")
	}
}

func TestBuildCommandArgs9pMount(t *testing.T) {
	plan := basePlan()
	plan.Mounts = []runtime.MountPlan{
		{Source: "/srv/dotfiles", Target: "/mnt/dotfiles", Driver: "9p", Tag: "yeast_mnt_0", ReadOnly: true},
		{Source: "/srv/code", Target: "/mnt/code", Driver: "9p", Tag: "yeast_mnt_1"},
	}
	got, err := BuildCommandArgs(plan)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"-fsdev", "local,id=fs0,path=/srv/dotfiles,security_model=mapped-xattr,readonly=on",
		"-device", "virtio-9p-pci,fsdev=fs0,mount_tag=yeast_mnt_0",
		"-fsdev", "local,id=fs1,path=/srv/code,security_model=mapped-xattr",
		"-device", "virtio-9p-pci,fsdev=fs1,mount_tag=yeast_mnt_1",
	}
	if !reflect.DeepEqual(tailArgs(got, len(want)), want) {
		t.Fatalf("9p args:\n got: %#v\nwant: %#v", tailArgs(got, len(want)), want)
	}
}

func TestBuildCommandArgsVirtiofsMount(t *testing.T) {
	plan := basePlan()
	plan.Mounts = []runtime.MountPlan{
		{Source: "/srv/dotfiles", Target: "/mnt/dotfiles", Driver: "virtiofs", Tag: "yeast_mnt_0", DaemonPath: "/usr/libexec/virtiofsd"},
	}
	got, err := BuildCommandArgs(plan)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"-object", "memory-backend-memfd,id=mem,size=2048M,share=on",
		"-numa", "node,memdev=mem",
		"-chardev", "socket,id=vfs0,path=/runtime/web/vfs-0.sock",
		"-device", "vhost-user-fs-pci,chardev=vfs0,tag=yeast_mnt_0",
	}
	if !reflect.DeepEqual(tailArgs(got, len(want)), want) {
		t.Fatalf("virtiofs args:\n got: %#v\nwant: %#v", tailArgs(got, len(want)), want)
	}
}

func TestManagementNetdevEgressModes(t *testing.T) {
	open := buildManagementNetdevArgFull("127.0.0.1", 2222, nil, nil)
	if strings.Contains(open, "restrict") {
		t.Fatalf("open mode must not restrict: %s", open)
	}
	offline := buildManagementNetdevArgFull("127.0.0.1", 2222, nil, &runtime.EgressPlan{Mode: "offline"})
	if !strings.Contains(offline, "restrict=on") {
		t.Fatalf("offline mode must add restrict=on: %s", offline)
	}
	allow := buildManagementNetdevArgFull("127.0.0.1", 2222, nil, &runtime.EgressPlan{Mode: "allowlist", ProxyPort: 43129})
	if !strings.Contains(allow, "restrict=on") || !strings.Contains(allow, "guestfwd=tcp:10.0.2.100:3128-tcp:127.0.0.1:43129") {
		t.Fatalf("allowlist mode missing restrict+guestfwd: %s", allow)
	}
	// SSH hostfwd must survive all modes
	if !strings.Contains(allow, "hostfwd=tcp:127.0.0.1:2222-:22") {
		t.Fatalf("ssh hostfwd lost: %s", allow)
	}
}

func TestSandboxArgs(t *testing.T) {
	args, err := BuildCommandArgs(sandboxTestPlan())
	if err != nil {
		t.Fatalf("BuildCommandArgs: %v", err)
	}
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "-sandbox on,obsolete=deny,elevateprivileges=deny,spawn=deny,resourcecontrol=deny") {
		t.Fatalf("missing hardened sandbox flags: %s", joined)
	}
	virgl := sandboxTestPlan()
	virgl.Display = &runtime.DisplayPlan{Type: "vnc", GPU: "virgl", Host: "127.0.0.1", Port: 5900}
	args2, err := BuildCommandArgs(virgl)
	if err != nil {
		t.Fatalf("virgl BuildCommandArgs: %v", err)
	}
	j2 := strings.Join(args2, " ")
	if strings.Contains(j2, "resourcecontrol=deny") {
		t.Fatalf("virgl must relax resourcecontrol: %s", j2)
	}
	if !strings.Contains(j2, "spawn=deny") {
		t.Fatalf("virgl still needs spawn=deny: %s", j2)
	}
}

func sandboxTestPlan() runtime.MachinePlan {
	return runtime.MachinePlan{
		Name: "web", RuntimeDir: "/runtime/web", LogPath: "/runtime/web/vm.log",
		MemoryMiB: 2048, CPUs: 2, SeedImagePath: "/runtime/web/seed.iso",
		Disk: runtime.DiskPlan{DiskPath: "/runtime/web/disk.qcow2"},
		Networks: runtime.NetworkPlan{Management: runtime.ManagementNetworkPlan{
			SSHHost: "127.0.0.1", SSHPort: 2222, InterfaceName: "yeastmgmt0", MACAddress: "52:54:00:11:22:33"}},
	}
}

func TestBootISOArgs(t *testing.T) {
	plan := sandboxTestPlan()
	plan.SeedImagePath = ""
	plan.BootISOPath = "/isos/live.iso"
	args, err := BuildCommandArgs(plan)
	if err != nil {
		t.Fatalf("BuildCommandArgs: %v", err)
	}
	j := strings.Join(args, " ")
	if !strings.Contains(j, "file=/isos/live.iso,if=ide,media=cdrom,readonly=on") {
		t.Fatalf("iso cdrom missing: %s", j)
	}
	if !strings.Contains(j, "-boot order=d") {
		t.Fatalf("boot order=d missing: %s", j)
	}
	// mutual exclusion
	plan.SeedImagePath = "/seed.iso"
	if _, err := BuildCommandArgs(plan); err == nil {
		t.Fatal("seed+bootiso must be rejected")
	}
}
