package app

import (
	"fmt"
	"testing"
)

func TestErrorHintNotFound(t *testing.T) {
	err := WrapError(ErrorCodeNotFound, "image ubuntu-24.04 not found in cache", nil)
	hint := ErrorHint(err)
	if hint == "" {
		t.Fatal("expected non-empty hint for image not found")
	}
	if got := ErrorHint(err); got == "" {
		t.Error("expected hint for image not found error")
	}
}

func TestErrorHintPreconditionNotRunning(t *testing.T) {
	err := WrapError(ErrorCodePrecondition, "instance web is not running", nil)
	hint := ErrorHint(err)
	if hint == "" {
		t.Fatal("expected non-empty hint for not running")
	}
}

func TestErrorHintInvalidArgumentPort(t *testing.T) {
	err := WrapError(ErrorCodeInvalidArgument, "ssh_port 2222 already in use", nil)
	hint := ErrorHint(err)
	if hint == "" {
		t.Fatal("expected non-empty hint for port conflict")
	}
}

func TestErrorHintProvisioningShell(t *testing.T) {
	err := WrapError(ErrorCodeProvisioning, "shell command failed: exit status 1", nil)
	hint := ErrorHint(err)
	if hint == "" {
		t.Fatal("expected non-empty hint for shell provision failure")
	}
}

func TestErrorHintGuestConnectionRefused(t *testing.T) {
	err := WrapError(ErrorCodeGuest, "dial tcp 127.0.0.1:2222: connection refused", nil)
	hint := ErrorHint(err)
	if hint == "" {
		t.Fatal("expected non-empty hint for connection refused")
	}
}

func TestErrorHintRuntimeKVM(t *testing.T) {
	err := WrapError(ErrorCodeRuntime, "KVM acceleration not available", nil)
	hint := ErrorHint(err)
	if hint == "" {
		t.Fatal("expected non-empty hint for KVM error")
	}
}

func TestErrorHintNilError(t *testing.T) {
	hint := ErrorHint(nil)
	if hint != "" {
		t.Errorf("expected empty hint for nil error, got: %q", hint)
	}
}

func TestErrorHintPlainError(t *testing.T) {
	hint := ErrorHint(fmt.Errorf("something went wrong"))
	if hint != "" {
		t.Errorf("expected empty hint for plain error, got: %q", hint)
	}
}

func TestErrorHintConflictLock(t *testing.T) {
	err := WrapError(ErrorCodeConflict, "state.lock: file already locked", nil)
	hint := ErrorHint(err)
	if hint == "" {
		t.Fatal("expected non-empty hint for lock conflict")
	}
}
