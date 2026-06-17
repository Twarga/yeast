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

func TestRenderHumanTemplateListResult(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := RenderHuman(&buf, "init", app.TemplateListResult{
		Templates: []app.TemplateSummary{
			{
				Name:        "caddy-single-vm",
				Description: "Ubuntu VM with Caddy provisioning.",
				Category:    "app",
				Source:      "builtin",
			},
		},
	})
	if err != nil {
		t.Fatalf("RenderHuman returned error: %v", err)
	}

	got := stripANSI(buf.String())
	for _, want := range []string{
		"Project templates",
		"NAME",
		"CATEGORY",
		"SOURCE",
		"DESCRIPTION",
		"caddy-single-vm",
		"app",
		"builtin",
		"Ubuntu VM with Caddy provisioning.",
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
			{Name: "web", Status: "running", ManagementIP: "0.0.0.0", SSHPort: 2222, LabIP: "10.10.10.10"},
			{Name: "api", Status: "stopped"},
		},
	})
	if err != nil {
		t.Fatalf("RenderHuman returned error: %v", err)
	}

	got := stripANSI(buf.String())
	for _, want := range []string{
		"Status",
		"NAME",
		"STATUS",
		"SSH",
		"LAB IP",
		"api",
		"stopped",
		"web",
		"running",
		"0.0.0.0:2222",
		"10.10.10.10",
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

func TestRenderHumanExecResult(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := RenderHuman(&buf, "exec", app.ExecResult{
		ProjectID: "proj_123",
		Instance:  "web",
		Run: app.GuestCommandResult{
			Command:  "whoami",
			ExitCode: 0,
			Stdout:   "yeast\n",
			Duration: 200 * time.Millisecond,
		},
	})
	if err != nil {
		t.Fatalf("RenderHuman returned error: %v", err)
	}

	got := stripANSI(buf.String())
	for _, want := range []string{
		"Command finished",
		"instance:",
		"web",
		"command:",
		"whoami",
		"exit_code:",
		"0",
		"stdout:",
		"yeast",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, got)
		}
	}
}

func TestRenderHumanCopyResult(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := RenderHuman(&buf, "copy", app.CopyResult{
		ProjectID:   "proj_123",
		Instance:    "web",
		Direction:   app.CopyFromGuest,
		Source:      "/tmp/report.txt",
		Destination: "/tmp/out.txt",
		Duration:    time.Second,
	})
	if err != nil {
		t.Fatalf("RenderHuman returned error: %v", err)
	}

	got := stripANSI(buf.String())
	for _, want := range []string{
		"Copy finished",
		"instance:",
		"web",
		"direction:",
		"from_guest",
		"source:",
		"/tmp/report.txt",
		"destination:",
		"/tmp/out.txt",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, got)
		}
	}
}

func TestRenderHumanInspectResult(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := RenderHuman(&buf, "inspect", app.InspectResult{
		ProjectID: "proj_123",
		Instance: app.StatusInstanceResult{
			Name:               "web",
			Status:             "running",
			ManagementIP:       "0.0.0.0",
			SSHPort:            2222,
			LabIP:              "10.10.10.10",
			RuntimeDir:         "/tmp/web",
			ProvisionLogPath:   "/tmp/web/provision.log",
			ProvisioningStatus: state.ProvisioningStatusReady,
		},
		SnapshotNames: []string{"clean", "post"},
		SnapshotCount: 2,
	})
	if err != nil {
		t.Fatalf("RenderHuman returned error: %v", err)
	}

	got := stripANSI(buf.String())
	for _, want := range []string{
		"Instance inspect",
		"name:",
		"web",
		"status:",
		"running",
		"ssh:",
		"0.0.0.0:2222",
		"lab_ip:",
		"10.10.10.10",
		"snapshot_count:",
		"2",
		"snapshots:",
		"clean, post",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, got)
		}
	}
}

func TestRenderHumanLogsResult(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := RenderHuman(&buf, "logs", app.LogsResult{
		ProjectID: "proj_123",
		Instance:  "web",
		LogPath:   "/tmp/web/vm.log",
		Content:   "booted\nready\n",
	})
	if err != nil {
		t.Fatalf("RenderHuman returned error: %v", err)
	}

	got := stripANSI(buf.String())
	for _, want := range []string{
		"Instance logs",
		"instance:",
		"web",
		"path:",
		"/tmp/web/vm.log",
		"content:",
		"booted",
		"ready",
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

func TestRenderHumanDoctorPlainDoesNotLeakANSI(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := RenderHuman(&buf, "doctor", app.DoctorResult{
		Environment: "container",
		SupportTier: "C",
		Checks: []app.DoctorCheck{
			{
				Name:    "qemu-system-x86_64",
				Status:  app.CheckStatusOK,
				Details: "/usr/bin/qemu-system-x86_64",
			},
		},
	})
	if err != nil {
		t.Fatalf("RenderHuman returned error: %v", err)
	}

	if ansiPattern.MatchString(buf.String()) {
		t.Fatalf("expected plain doctor output to avoid ANSI escapes, got:\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), "Tier C") {
		t.Fatalf("expected doctor output to include support tier text, got:\n%s", buf.String())
	}
}

func stripANSI(value string) string {
	return ansiPattern.ReplaceAllString(value, "")
}
