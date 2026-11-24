package output

import (
	"encoding/json"
	"testing"
)

// The yeast.v2 envelope/event schemas are additive over v1. These fixtures
// pin the new fields so regressions are caught.
func TestV2EnvelopeFixtures(t *testing.T) {
	envelopeSchema := loadSchemaFile(t, "../yeast.v2/envelope.schema.json")
	defs := schemaDefs(envelopeSchema)

	success := map[string]any{
		"ok":             true,
		"schema_version": "yeast.v2",
		"command":        "fork",
		"data": map[string]any{
			"project_id": "p1", "source": "web",
			"forks": []any{map[string]any{"name": "web-fork-1"}},
		},
		"meta": map[string]any{
			"actor": "mcp:claude-code", "duration_ms": 42, "project_id": "p1",
		},
	}
	if errs := validateAgainst(envelopeSchema, jsonRoundTrip(t, success), "$", defs); len(errs) > 0 {
		t.Fatalf("v2 success fixture invalid: %v", errs)
	}

	errDoc := map[string]any{
		"ok":             false,
		"schema_version": "yeast.v2",
		"error": map[string]any{
			"code": "failed_precondition", "message": "no KVM",
			"retryable": false,
			"recovery":  []any{"enable virtualization in BIOS", "yeast doctor"},
		},
	}
	if errs := validateAgainst(envelopeSchema, jsonRoundTrip(t, errDoc), "$", defs); len(errs) > 0 {
		t.Fatalf("v2 error fixture invalid: %v", errs)
	}

	// v2 rejects a v1-tagged document
	bad := map[string]any{"ok": true, "schema_version": "yeast.v1", "command": "x"}
	if errs := validateAgainst(envelopeSchema, jsonRoundTrip(t, bad), "$", defs); len(errs) == 0 {
		t.Fatal("v1-tagged doc passed v2 schema")
	}
}

// jsonRoundTrip normalizes a fixture through the real JSON codec so numbers
// arrive as float64 like a decoded wire document.
func jsonRoundTrip(t *testing.T, v any) any {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var out any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestV2EventFixtures(t *testing.T) {
	eventSchema := loadSchemaFile(t, "../yeast.v2/event.schema.json")
	defs := schemaDefs(eventSchema)

	ev := map[string]any{
		"schema_version": "yeast.v2", "type": "event",
		"name": "fork.created", "command": "fork",
		"project_id": "p1", "instance": "web",
		"time": "2026-11-23T12:00:00Z",
		"meta": map[string]any{"actor": "api"},
	}
	if errs := validateAgainst(eventSchema, ev, "$", defs); len(errs) > 0 {
		t.Fatalf("v2 event fixture invalid: %v", errs)
	}
}
