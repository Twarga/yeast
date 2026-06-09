package qemu

import (
	"net/netip"
	"reflect"
	"testing"
	"yeast/internal/runtime"
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
		"-enable-kvm",
		"-name", "web",
		"-m", "2048",
		"-smp", "2",
		"-drive", "file=/runtime/web/disk.qcow2,if=virtio,format=qcow2",
		"-drive", "file=/runtime/web/seed.iso,if=virtio,media=cdrom,readonly=on",
		"-netdev", "user,id=mgmt0,hostfwd=tcp:127.0.0.1:2222-:22",
		"-device", "virtio-net-pci,netdev=mgmt0,mac=52:54:00:11:22:33",
		"-qmp", "unix:/runtime/web/qmp.sock,server,nowait",
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
		"-enable-kvm",
		"-name", "target",
		"-m", "1024",
		"-smp", "1",
		"-drive", "file=/runtime/proj/instances/target/disk.qcow2,if=virtio,format=qcow2",
		"-drive", "file=/runtime/proj/instances/target/seed.iso,if=virtio,media=cdrom,readonly=on",
		"-netdev", "user,id=mgmt0,hostfwd=tcp:127.0.0.1:2222-:22",
		"-device", "virtio-net-pci,netdev=mgmt0,mac=52:54:00:11:22:33",
		"-qmp", "unix:/runtime/proj/instances/target/qmp.sock,server,nowait",
		"-netdev", wantLabNetdev,
		"-device", "virtio-net-pci,netdev=lab0,mac=52:54:00:aa:bb:cc",
		"-nographic",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected args:\n got: %#v\nwant: %#v", got, want)
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
