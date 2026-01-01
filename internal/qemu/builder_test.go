package qemu

import (
	"reflect"
	"testing"
	"yeast/internal/types"
)

func TestBuildArguments(t *testing.T) {
	params := Params{
		Machine: types.Machine{
			Name: "test-vm",
			Specs: types.Specs{
				CPUs:   2,
				Memory: "2G",
			},
		},
		ImagePath:   "/path/to/image.qcow2",
		SeedISOPath: "/path/to/seed.iso",
		SSHPort:     2222,
	}

	expected := []string{
		"-enable-kvm",
		"-m", "2G",
		"-smp", "2",
		"-drive", "file=/path/to/image.qcow2,format=qcow2,if=virtio",
		"-drive", "file=/path/to/seed.iso,format=raw,if=virtio",
		"-netdev", "user,id=net0,hostfwd=tcp::2222-:22",
		"-device", "virtio-net-pci,netdev=net0",
		"-nographic",
	}

	actual := BuildArguments(params)

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("BuildArguments() = %v, want %v", actual, expected)
	}
}
