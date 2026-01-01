package qemu

import (
	"fmt"
	"yeast/internal/types"
)

// Params contains all information needed to construct a QEMU command
type Params struct {
	Machine     types.Machine
	ImagePath   string // Path to the project-local overlay image
	SeedISOPath string // Path to the cloud-init seed ISO
	SSHPort     int    // Host port to forward to VM port 22
}

// BuildArguments constructs the slice of strings for exec.Command
func BuildArguments(p Params) []string {
	args := []string{
		"-enable-kvm",
		"-m", p.Machine.Specs.Memory,
		"-smp", fmt.Sprintf("%d", p.Machine.Specs.CPUs),
		"-drive", fmt.Sprintf("file=%s,format=qcow2,if=virtio", p.ImagePath),
		"-drive", fmt.Sprintf("file=%s,format=raw,if=virtio", p.SeedISOPath),
		"-netdev", fmt.Sprintf("user,id=net0,hostfwd=tcp::%d-:22", p.SSHPort),
		"-device", "virtio-net-pci,netdev=net0",
		"-nographic",
	}

	return args
}
