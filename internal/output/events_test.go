package output

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
	"yeast/internal/app"
)

func TestRenderJSONEvent(t *testing.T) {
	t.Parallel()

	event := app.NewEvent("up", app.EventVMStarting, app.EventOptions{
		ProjectID: "proj_123",
		Instance:  "web",
		Now:       time.Date(2026, 5, 26, 13, 10, 0, 0, time.UTC),
	})

	var buf bytes.Buffer
	if err := RenderJSONEvent(&buf, event); err != nil {
		t.Fatalf("RenderJSONEvent returned error: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal event: %v\npayload: %s", err, buf.String())
	}
	if payload["schema_version"] != SchemaVersion {
		t.Fatalf("unexpected schema_version: %#v", payload["schema_version"])
	}
	if payload["type"] != "event" {
		t.Fatalf("unexpected type: %#v", payload["type"])
	}
	if payload["name"] != "vm.starting" {
		t.Fatalf("unexpected event name: %#v", payload["name"])
	}
}
