package output

import (
	"bytes"
	"strings"
	"testing"
	"yeast/internal/app"
)

func TestRenderHumanInitResult(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := RenderHuman(&buf, "init", app.InitResult{
		ConfigPath:   "/tmp/project/yeast.yaml",
		MetadataPath: "/tmp/project/.yeast/project.json",
		ProjectID:    "proj_123",
	})
	if err != nil {
		t.Fatalf("RenderHuman returned error: %v", err)
	}

	got := buf.String()
	for _, want := range []string{
		"Created /tmp/project/yeast.yaml",
		"Created /tmp/project/.yeast/project.json",
		"Project ID: proj_123",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, got)
		}
	}
}

func TestRenderHumanStatusResult(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := RenderHuman(&buf, "status", app.StatusResult{
		Instances: []app.StatusInstanceResult{
			{Name: "web", Status: "running", SSHPort: 2222},
			{Name: "api", Status: "stopped"},
		},
	})
	if err != nil {
		t.Fatalf("RenderHuman returned error: %v", err)
	}

	got := buf.String()
	want := "api\tstopped\nweb\trunning\t127.0.0.1:2222\n"
	if got != want {
		t.Fatalf("unexpected output:\n got: %q\nwant: %q", got, want)
	}
}
