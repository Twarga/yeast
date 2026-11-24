package app

import (
	"errors"
	"fmt"
	"os"
)

// checkKVMReadiness is the production pre-flight for Service.Up.
// QEMU always launches with `-machine q35,accel=kvm -cpu host -enable-kvm`,
// so a missing or unreadable /dev/kvm guarantees a failed boot.
// It verifies the device node exists AND that the current user can open it
// read/write, mirroring the `yeast doctor` check.
func checkKVMReadiness() error {
	info, err := os.Stat("/dev/kvm")
	if err != nil {
		return fmt.Errorf("/dev/kvm is not available on this host: %w", err)
	}
	if info.Mode()&os.ModeDevice == 0 {
		return errors.New("/dev/kvm exists but is not a device node")
	}
	f, err := os.OpenFile("/dev/kvm", os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("/dev/kvm is not readable/writable by the current user: %w", err)
	}
	return f.Close()
}

// kvmPreconditionError wraps a readiness failure as a typed
// failed_precondition error with environment-aware recovery steps.
func kvmPreconditionError(err error) error {
	steps := []string{
		"Run `yeast doctor` to diagnose host virtualization support.",
	}
	switch {
	case errors.Is(err, os.ErrPermission):
		steps = append(steps,
			"Run `sudo usermod -aG kvm $USER`, then log out and back in (or `newgrp kvm`).",
			"Verify with `ls -l /dev/kvm` — the kvm group needs read/write access.",
		)
	case errors.Is(err, os.ErrNotExist):
		steps = append(steps,
			"Native Linux: enable Intel VT-x / AMD-V in BIOS/UEFI, then `sudo modprobe kvm_intel` (or kvm_amd).",
			"WSL2: add `nestedVirtualization=true` under [wsl2] in %UserProfile%\\.wslconfig on Windows, run `wsl --shutdown`, then reopen.",
			"Containers: pass `--device /dev/kvm`, or run yeast on the host.",
		)
	default:
		steps = append(steps,
			"Ensure /dev/kvm is a real KVM device node owned by group kvm with read/write permission.",
		)
	}
	return WithRecovery(
		WrapError(ErrorCodePrecondition, fmt.Sprintf("KVM is required to start VMs: %v", err), err),
		steps...,
	)
}
