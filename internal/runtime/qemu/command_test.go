package qemu

import (
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
				SSHHost: "127.0.0.1",
				SSHPort: 2222,
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
		"-device", "virtio-net-pci,netdev=mgmt0",
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
				SSHHost: "127.0.0.1",
				SSHPort: 2201,
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
						SSHHost: "127.0.0.1",
						SSHPort: 2222,
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
						SSHHost: "127.0.0.1",
						SSHPort: 2222,
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
						SSHHost: "127.0.0.1",
						SSHPort: 2222,
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
