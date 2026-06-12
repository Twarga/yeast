package app

import (
	"strings"
)

// ErrorHint returns a user-friendly recovery hint for the given error.
// Returns "" if no hint is available.
func ErrorHint(err error) string {
	if err == nil {
		return ""
	}

	appErr, ok := err.(*AppError)
	if !ok {
		return ""
	}

	msg := strings.ToLower(appErr.Message)

	switch appErr.Code {
	case ErrorCodeNotFound:
		if strings.Contains(msg, "image") {
			return `Run "yeast pull --list" to see available images.`
		}
		if strings.Contains(msg, "project") || strings.Contains(msg, "metadata") {
			return `Initialize a project first: yeast init`
		}
		if strings.Contains(msg, "snapshot") {
			return `Run "yeast snapshots <instance>" to list available snapshots.`
		}

	case ErrorCodePrecondition:
		if strings.Contains(msg, "ssh") {
			return `Try: yeast down && yeast up
Check VM logs: yeast logs <instance>`
		}
		if strings.Contains(msg, "not running") || strings.Contains(msg, "is stopped") {
			return `Start instances with: yeast up`
		}
		if strings.Contains(msg, "metadata") || strings.Contains(msg, "not a yeast project") {
			return `Initialize a project first: yeast init`
		}
		if strings.Contains(msg, "running") && strings.Contains(msg, "already") {
			return `The instance is already running. Use "yeast status" to check.`
		}

	case ErrorCodeInvalidArgument:
		if strings.Contains(msg, "ssh_port") || strings.Contains(msg, "port") {
			return `Another instance may be using this port. Try: yeast status`
		}
		if strings.Contains(msg, "image") {
			return `Run "yeast pull --list" to see available images.`
		}
		if strings.Contains(msg, "no-provision") && strings.Contains(msg, "reprovision") {
			return `Use either --no-provision or --reprovision, not both.`
		}

	case ErrorCodeConflict:
		if strings.Contains(msg, "lock") || strings.Contains(msg, "busy") {
			return `Another yeast operation may be running. Wait or remove .yeast/state.lock.`
		}

	case ErrorCodeProvisioning:
		if strings.Contains(msg, "package") {
			return `Check the provision log: yeast logs <instance>
The package may not be available in the VM's repository.`
		}
		if strings.Contains(msg, "file") {
			return `Verify the source file exists in the project directory.`
		}
		if strings.Contains(msg, "shell") {
			return `Check the provision log: yeast logs <instance>
The command may have failed inside the VM.`
		}

	case ErrorCodeGuest:
		if strings.Contains(msg, "connection refused") {
			return `The VM may not be running. Try: yeast status
If running, wait a few seconds and retry.`
		}
		if strings.Contains(msg, "timeout") || strings.Contains(msg, "timed out") {
			return `The operation took too long. The VM may be overloaded.
Try: yeast down && yeast up`
		}

	case ErrorCodeRuntime:
		if strings.Contains(msg, "kvm") || strings.Contains(msg, "kvm acceleration") {
			return `KVM is not available. Check: yeast doctor
Ensure /dev/kvm exists and your user has permission (add to "kvm" group).`
		}
		if strings.Contains(msg, "qemu") {
			return `QEMU may not be installed. Check: yeast doctor
Install: sudo apt install qemu-system-x86`
		}
		if strings.Contains(msg, "disk") || strings.Contains(msg, "space") {
			return `Check available disk space: df -h ~/.yeast`
		}
	}

	return ""
}
