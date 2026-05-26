package app

import (
	"errors"
	"testing"
)

func TestErrorCodeValuesAreStable(t *testing.T) {
	t.Parallel()

	tests := map[ErrorCode]string{
		ErrorCodeUnknown:         "unknown",
		ErrorCodeInvalidArgument: "invalid_argument",
		ErrorCodeNotFound:        "not_found",
		ErrorCodeConflict:        "conflict",
		ErrorCodePrecondition:    "failed_precondition",
		ErrorCodeTimeout:         "timeout",
		ErrorCodeRuntime:         "runtime_error",
		ErrorCodeProvisioning:    "provisioning_failed",
		ErrorCodeGuest:           "guest_error",
		ErrorCodeInternal:        "internal",
	}

	for code, want := range tests {
		if string(code) != want {
			t.Fatalf("unexpected value for %s: got %q want %q", want, code, want)
		}
	}
}

func TestWrapErrorPreservesCodeAndCause(t *testing.T) {
	t.Parallel()

	cause := errors.New("boom")
	err := WrapError(ErrorCodeConflict, "state lock busy", cause)

	if err.Code != ErrorCodeConflict {
		t.Fatalf("unexpected code: %q", err.Code)
	}
	if err.Unwrap() != cause {
		t.Fatal("expected wrapped cause")
	}
}

func TestNormalizeErrorWrapsGenericError(t *testing.T) {
	t.Parallel()

	err := NormalizeError(errors.New("plain failure"))
	if err.Code != ErrorCodeUnknown {
		t.Fatalf("unexpected code: %q", err.Code)
	}
	if err.Message != "plain failure" {
		t.Fatalf("unexpected message: %q", err.Message)
	}
}
