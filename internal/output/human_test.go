package output

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
	"time"
	"yeast/internal/app"
	"yeast/internal/state"
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

func TestRenderHumanProvisionResult(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := RenderHuman(&buf, "provision", app.ProvisionResult{
		ProjectID: "proj_123",
		Instance: app.ProvisionInstanceResult{
			Name:               "web",
			ProvisioningStatus: state.ProvisioningStatusReady,
			SSHAddress:         "127.0.0.1:2205",
			ProvisionLogPath:   "/tmp/provision.log",
		},
	})
	if err != nil {
		t.Fatalf("RenderHuman returned error: %v", err)
	}

	got := stripANSI(buf.String())
	for _, want := range []string{
		"Provisioning finished",
		"instance:",
		"web",
		"status:",
		"provisioned",
		"ssh:",
		"127.0.0.1:2205",
		"log:",
		"/tmp/provision.log",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, got)
		}
	}
}

func TestRenderHumanSnapshotResult(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := RenderHuman(&buf, "snapshot", app.SnapshotResult{
		ProjectID: "proj_123",
		Instance:  "web",
		Snapshot: state.SnapshotState{
			Name:        "clean",
			CreatedAt:   time.Date(2026, 5, 22, 10, 30, 0, 0, time.UTC),
			Description: "Ready baseline",
			DiskPath:    "/tmp/web/snapshots/clean.qcow2",
		},
	})
	if err != nil {
		t.Fatalf("RenderHuman returned error: %v", err)
	}

	got := stripANSI(buf.String())
	for _, want := range []string{
		"Snapshot created",
		"instance:",
		"web",
		"snapshot:",
		"clean",
		"description:",
		"Ready baseline",
		"path:",
		"/tmp/web/snapshots/clean.qcow2",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, got)
		}
	}
}

func TestRenderHumanRestoreResult(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := RenderHuman(&buf, "restore", app.RestoreResult{
		ProjectID: "proj_123",
		Instance:  "web",
		Snapshot: state.SnapshotState{
			Name:     "clean",
			DiskPath: "/tmp/web/snapshots/clean.qcow2",
		},
	})
	if err != nil {
		t.Fatalf("RenderHuman returned error: %v", err)
	}

	got := stripANSI(buf.String())
	for _, want := range []string{
		"Snapshot restored",
		"instance:",
		"web",
		"snapshot:",
		"clean",
		"path:",
		"/tmp/web/snapshots/clean.qcow2",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, got)
		}
	}
}

func TestRenderHumanSnapshotsResult(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := RenderHuman(&buf, "snapshots", app.SnapshotsResult{
		ProjectID: "proj_123",
		Instance:  "web",
		Snapshots: []state.SnapshotState{
			{
				Name:        "clean",
				CreatedAt:   time.Date(2026, 5, 22, 10, 30, 0, 0, time.UTC),
				Description: "Ready baseline",
				DiskPath:    "/tmp/web/snapshots/clean.qcow2",
			},
		},
	})
	if err != nil {
		t.Fatalf("RenderHuman returned error: %v", err)
	}

	got := stripANSI(buf.String())
	for _, want := range []string{
		"Instance snapshots",
		"instance:",
		"web",
		"NAME",
		"CREATED",
		"DESCRIPTION",
		"PATH",
		"clean",
		"Ready baseline",
		"/tmp/web/snapshots/clean.qcow2",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, got)
		}
	}
}

func TestRenderHumanDeleteSnapshotResult(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := RenderHuman(&buf, "delete-snapshot", app.DeleteSnapshotResult{
		ProjectID: "proj_123",
		Instance:  "web",
		Snapshot:  "clean",
	})
	if err != nil {
		t.Fatalf("RenderHuman returned error: %v", err)
	}

	got := stripANSI(buf.String())
	for _, want := range []string{
		"Snapshot deleted",
		"instance:",
		"web",
		"snapshot:",
		"clean",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, got)
		}
	}
}

func stripANSI(value string) string {
	return ansiPattern.ReplaceAllString(value, "")
}
