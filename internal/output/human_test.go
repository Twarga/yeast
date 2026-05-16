package output

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
	"yeast/internal/app"
)

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;:]*m`)

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

	got := stripANSI(buf.String())
	for _, want := range []string{
		"Project initialized",
		"config:",
		"/tmp/project/yeast.yaml",
		"metadata:",
		"/tmp/project/.yeast/project.json",
		"project:",
		"proj_123",
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

	got := stripANSI(buf.String())
	for _, want := range []string{
		"Project status",
		"NAME",
		"STATUS",
		"SSH",
		"api",
		"stopped",
		"web",
		"running",
		"127.0.0.1:2222",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, got)
		}
	}
}

func stripANSI(value string) string {
	return ansiPattern.ReplaceAllString(value, "")
}
