package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
	"yeast/internal/app"
)

func TestRenderJSONSuccess(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := RenderJSONSuccess(&buf, "version", map[string]any{"value": "0.0.0-dev"})
	if err != nil {
		t.Fatalf("RenderJSONSuccess returned error: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal json: %v", err)
	}
	if got["ok"] != true {
		t.Fatalf("expected ok=true, got %#v", got["ok"])
	}
	if got["schema_version"] != SchemaVersion {
		t.Fatalf("unexpected schema_version: %#v", got["schema_version"])
	}
	if got["command"] != "version" {
		t.Fatalf("unexpected command: %#v", got["command"])
	}
}

func TestRenderJSONError(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := RenderJSONError(&buf, app.WrapError(app.ErrorCodeConflict, "state lock busy", errors.New("busy")))
	if err != nil {
		t.Fatalf("RenderJSONError returned error: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal json: %v", err)
	}
	if got["ok"] != false {
		t.Fatalf("expected ok=false, got %#v", got["ok"])
	}
	if got["schema_version"] != SchemaVersion {
		t.Fatalf("unexpected schema_version: %#v", got["schema_version"])
	}
	errorBody, ok := got["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error object, got %#v", got["error"])
	}
	if errorBody["code"] != "conflict" {
		t.Fatalf("unexpected code: %#v", errorBody["code"])
	}
	if errorBody["message"] != "state lock busy" {
		t.Fatalf("unexpected message: %#v", errorBody["message"])
	}
}
