package main

import (
	"bytes"
	"encoding/json"
	"testing"
	"yeast/internal/app"
)

func TestPullUnsupportedImageRendersInvalidArgumentJSON(t *testing.T) {
	t.Parallel()

	previous := outputJSON
	outputJSON = true
	defer func() {
		outputJSON = previous
	}()

	root := newRootCmd(app.NewService())
	root.SetArgs([]string{"pull", "does-not-exist", "--json"})

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if renderErr := renderCommandError(&buf, err); renderErr != nil {
		t.Fatalf("renderCommandError returned error: %v", renderErr)
	}

	var payload map[string]any
	if decodeErr := json.Unmarshal(buf.Bytes(), &payload); decodeErr != nil {
		t.Fatalf("unmarshal rendered json: %v\npayload: %s", decodeErr, buf.String())
	}

	errorBody, ok := payload["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error body, got %#v", payload["error"])
	}
	if errorBody["code"] != "invalid_argument" {
		t.Fatalf("expected invalid_argument code, got %#v", errorBody["code"])
	}
}
