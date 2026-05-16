package app

import (
	"errors"
	"testing"
)

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
