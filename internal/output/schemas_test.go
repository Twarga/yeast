package output

import "testing"

func TestSuccessEnvelopeDefaults(t *testing.T) {
	t.Parallel()

	envelope := SuccessEnvelope{
		OK:      true,
		Command: "up",
		Data: map[string]any{
			"project_id": "proj_123",
		},
	}

	if !envelope.OK {
		t.Fatal("expected success envelope ok=true")
	}
	if envelope.Command != "up" {
		t.Fatalf("unexpected command: %q", envelope.Command)
	}
}

func TestErrorEnvelopeDefaults(t *testing.T) {
	t.Parallel()

	envelope := ErrorEnvelope{
		OK: false,
		Error: ErrorBody{
			Code:    "invalid_argument",
			Message: "missing project root",
		},
	}

	if envelope.OK {
		t.Fatal("expected error envelope ok=false")
	}
	if envelope.Error.Code != "invalid_argument" {
		t.Fatalf("unexpected code: %q", envelope.Error.Code)
	}
}
