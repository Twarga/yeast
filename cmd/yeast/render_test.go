package main

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"
	"yeast/internal/app"
	"yeast/internal/state"
)

func TestRenderCommandOutputJSONForCoreCommands(t *testing.T) {
	tests := []struct {
		name         string
		command      string
		data         any
		requiredKeys []string
	}{
		{
			name:    "init",
			command: "init",
			data: app.InitResult{
				ConfigPath:   "/tmp/yeast.yaml",
				MetadataPath: "/tmp/.yeast/project.json",
				ProjectID:    "proj_123",
				Template:     "ubuntu-basic",
			},
			requiredKeys: []string{"project_id", "config_path", "metadata_path", "template"},
		},
		{
			name:    "template-list",
			command: "init",
			data: app.TemplateListResult{
				Templates: []app.TemplateSummary{
					{
						Name:        "ubuntu-basic",
						Title:       "Ubuntu Basic",
						Description: "Minimal Ubuntu VM starter.",
						Category:    "vm",
						Source:      "builtin",
					},
				},
			},
			requiredKeys: []string{"templates", "templates.0.name", "templates.0.source"},
		},
		{
			name:    "pull",
			command: "pull",
			data: app.PullResult{
				ImageName: "ubuntu-24.04",
				ImagePath: "/tmp/cache/images/ubuntu-24.04/image.qcow2",
			},
			requiredKeys: []string{"image_name", "image_path"},
		},
		{
			name:    "doctor",
			command: "doctor",
			data: app.DoctorResult{
				Blockers: 1,
				Warnings: 1,
				Checks: []app.DoctorCheck{
					{Name: "qemu-system-x86_64", Status: app.CheckStatusBlocker, Details: "required"},
					{Name: "cache-directory", Status: app.CheckStatusWarning, Details: "missing"},
				},
			},
			requiredKeys: []string{"checks", "checks.0.name", "blockers", "warnings"},
		},
		{
			name:    "up",
			command: "up",
			data: app.UpResult{
				ProjectID: "proj_123",
				Instances: []app.UpInstanceResult{
					{Name: "web", Status: "running", SSHAddress: "127.0.0.1:2222", SSHPort: 2222},
				},
			},
			requiredKeys: []string{"project_id", "instances", "instances.0.name", "instances.0.ssh_port", "instances.0.ssh_address"},
		},
		{
			name:    "status",
			command: "status",
			data: app.StatusResult{
				ProjectID: "proj_123",
				Instances: []app.StatusInstanceResult{
					{Name: "web", Status: "running", SSHPort: 2222, User: "yeast", LabIP: "10.10.10.10"},
				},
			},
			requiredKeys: []string{"project_id", "instances", "instances.0.name", "instances.0.ssh_port", "instances.0.user", "instances.0.lab_ip"},
		},
		{
			name:    "provision",
			command: "provision",
			data: app.ProvisionResult{
				ProjectID: "proj_123",
				Instance: app.ProvisionInstanceResult{
					Name:               "web",
					ProvisioningStatus: "provisioned",
					SSHAddress:         "127.0.0.1:2222",
					SSHPort:            2222,
					ProvisionLogPath:   "/tmp/provision.log",
				},
			},
			requiredKeys: []string{"project_id", "instance", "instance.name", "instance.provisioning_status", "instance.provision_log_path"},
		},
		{
			name:    "exec",
			command: "exec",
			data: app.ExecResult{
				ProjectID: "proj_123",
				Instance:  "web",
				Run: app.GuestCommandResult{
					Command:    "whoami",
					ExitCode:   0,
					Stdout:     "yeast\n",
					Stderr:     "",
					StartedAt:  time.Date(2026, 5, 25, 11, 0, 0, 0, time.UTC),
					FinishedAt: time.Date(2026, 5, 25, 11, 0, 0, 200000000, time.UTC),
					Duration:   200 * time.Millisecond,
				},
			},
			requiredKeys: []string{"project_id", "instance", "run", "run.command", "run.exit_code", "run.stdout", "run.timed_out"},
		},
		{
			name:    "copy",
			command: "copy",
			data: app.CopyResult{
				ProjectID:   "proj_123",
				Instance:    "web",
				Direction:   app.CopyToGuest,
				Source:      "/tmp/site.txt",
				Destination: "/home/yeast/site.txt",
				StartedAt:   time.Date(2026, 5, 25, 11, 1, 0, 0, time.UTC),
				FinishedAt:  time.Date(2026, 5, 25, 11, 1, 1, 0, time.UTC),
				Duration:    time.Second,
			},
			requiredKeys: []string{"project_id", "instance", "direction", "source", "destination"},
		},
		{
			name:    "inspect",
			command: "inspect",
			data: app.InspectResult{
				ProjectID: "proj_123",
				Instance: app.StatusInstanceResult{
					Name:               "web",
					Status:             "running",
					SSHPort:            2222,
					User:               "yeast",
					LabIP:              "10.10.10.10",
					RuntimeDir:         "/tmp/web",
					ProvisionLogPath:   "/tmp/web/provision.log",
					ProvisioningStatus: state.ProvisioningStatusReady,
				},
				SnapshotNames: []string{"clean"},
				SnapshotCount: 1,
			},
			requiredKeys: []string{"project_id", "instance", "instance.name", "instance.user", "instance.provisioning_status", "snapshot_names", "snapshot_count"},
		},
		{
			name:    "logs",
			command: "logs",
			data: app.LogsResult{
				ProjectID: "proj_123",
				Instance:  "web",
				LogPath:   "/tmp/web/vm.log",
				Content:   "booted\n",
			},
			requiredKeys: []string{"project_id", "instance", "log_path", "content"},
		},
		{
			name:    "snapshot",
			command: "snapshot",
			data: app.SnapshotResult{
				ProjectID: "proj_123",
				Instance:  "web",
				Snapshot: state.SnapshotState{
					Name:      "clean",
					CreatedAt: time.Date(2026, 5, 22, 10, 30, 0, 0, time.UTC),
					DiskPath:  "/tmp/web/snapshots/clean.qcow2",
				},
			},
			requiredKeys: []string{"project_id", "instance", "snapshot", "snapshot.name"},
		},
		{
			name:    "restore",
			command: "restore",
			data: app.RestoreResult{
				ProjectID: "proj_123",
				Instance:  "web",
				Snapshot: state.SnapshotState{
					Name:     "clean",
					DiskPath: "/tmp/web/snapshots/clean.qcow2",
				},
			},
			requiredKeys: []string{"project_id", "instance", "snapshot", "snapshot.name"},
		},
		{
			name:    "snapshots",
			command: "snapshots",
			data: app.SnapshotsResult{
				ProjectID: "proj_123",
				Instance:  "web",
				Snapshots: []state.SnapshotState{
					{
						Name:      "clean",
						CreatedAt: time.Date(2026, 5, 22, 10, 30, 0, 0, time.UTC),
						DiskPath:  "/tmp/web/snapshots/clean.qcow2",
					},
				},
			},
			requiredKeys: []string{"project_id", "instance", "snapshots", "snapshots.0.name"},
		},
		{
			name:    "delete-snapshot",
			command: "delete-snapshot",
			data: app.DeleteSnapshotResult{
				ProjectID: "proj_123",
				Instance:  "web",
				Snapshot:  "clean",
			},
			requiredKeys: []string{"project_id", "instance", "snapshot"},
		},
		{
			name:    "down",
			command: "down",
			data: app.DownResult{
				ProjectID: "proj_123",
				Instances: []app.DownInstanceResult{
					{Name: "web", Status: "stopped"},
				},
			},
			requiredKeys: []string{"project_id", "instances", "instances.0.name", "instances.0.status"},
		},
		{
			name:    "destroy",
			command: "destroy",
			data: app.DestroyResult{
				ProjectID: "proj_123",
				Instances: []app.DestroyInstanceResult{
					{Name: "web", Status: "destroyed"},
				},
			},
			requiredKeys: []string{"project_id", "instances", "instances.0.name", "instances.0.status"},
		},
		{
			name:    "version",
			command: "version",
			data:    "0.0.0-dev",
		},
	}

	previous := outputJSON
	outputJSON = true
	defer func() {
		outputJSON = previous
	}()

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := renderCommandOutput(&buf, tt.command, tt.data); err != nil {
				t.Fatalf("renderCommandOutput returned error: %v", err)
			}

			var payload map[string]any
			if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
				t.Fatalf("unmarshal rendered json: %v\npayload: %s", err, buf.String())
			}

			if payload["ok"] != true {
				t.Fatalf("expected ok=true, got %#v", payload["ok"])
			}
			if payload["command"] != tt.command {
				t.Fatalf("expected command %q, got %#v", tt.command, payload["command"])
			}
			data, ok := payload["data"]
			if !ok {
				t.Fatalf("expected data field, payload=%#v", payload)
			}
			if got := payload["schema_version"]; got != "yeast.v1" {
				t.Fatalf("expected schema_version yeast.v1, got %#v", got)
			}
			for _, key := range tt.requiredKeys {
				if !jsonPathExists(data, key) {
					t.Fatalf("expected data key %q in payload: %#v", key, data)
				}
			}
		})
	}
}

func jsonPathExists(value any, path string) bool {
	current := value
	for _, token := range strings.Split(path, ".") {
		switch typed := current.(type) {
		case map[string]any:
			next, ok := typed[token]
			if !ok {
				return false
			}
			current = next
		case []any:
			index, err := strconv.Atoi(token)
			if err != nil || index < 0 || index >= len(typed) {
				return false
			}
			current = typed[index]
		default:
			return false
		}
	}
	return true
}

func TestRenderCommandErrorJSON(t *testing.T) {
	t.Parallel()

	previous := outputJSON
	outputJSON = true
	defer func() {
		outputJSON = previous
	}()

	var buf bytes.Buffer
	err := renderCommandError(&buf, app.WrapError(app.ErrorCodeConflict, "state lock busy", nil))
	if err != nil {
		t.Fatalf("renderCommandError returned error: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal rendered json: %v\npayload: %s", err, buf.String())
	}
	if payload["ok"] != false {
		t.Fatalf("expected ok=false, got %#v", payload["ok"])
	}
	errorBody, ok := payload["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error body, got %#v", payload["error"])
	}
	if errorBody["code"] != "conflict" {
		t.Fatalf("expected conflict code, got %#v", errorBody["code"])
	}
	if errorBody["message"] != "state lock busy" {
		t.Fatalf("expected message, got %#v", errorBody["message"])
	}
}

func TestEventSinkRequiresJSON(t *testing.T) {
	previousJSON := outputJSON
	previousEvents := outputEvents
	outputJSON = false
	outputEvents = true
	defer func() {
		outputJSON = previousJSON
		outputEvents = previousEvents
	}()

	_, err := eventSink(&bytes.Buffer{})
	if err == nil {
		t.Fatal("expected --events without --json to fail")
	}
}

func TestDownAndDestroyEventsRequireJSON(t *testing.T) {
	tests := []string{"down", "destroy"}

	for _, command := range tests {
		command := command
		t.Run(command, func(t *testing.T) {
			previousJSON := outputJSON
			previousEvents := outputEvents
			outputJSON = false
			outputEvents = false
			defer func() {
				outputJSON = previousJSON
				outputEvents = previousEvents
			}()

			root := newRootCmd(app.NewService())
			root.SetArgs([]string{command, "--events"})

			var buf bytes.Buffer
			root.SetOut(&buf)
			root.SetErr(&buf)

			err := root.Execute()
			if err == nil {
				t.Fatal("expected --events without --json to fail")
			}
			if !strings.Contains(err.Error(), "--events requires --json") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestEventSinkRendersJSONLine(t *testing.T) {
	previousJSON := outputJSON
	previousEvents := outputEvents
	outputJSON = true
	outputEvents = true
	defer func() {
		outputJSON = previousJSON
		outputEvents = previousEvents
	}()

	var buf bytes.Buffer
	sink, err := eventSink(&buf)
	if err != nil {
		t.Fatalf("eventSink returned error: %v", err)
	}
	if sink == nil {
		t.Fatal("expected event sink")
	}

	sink(app.NewEvent("up", app.EventSSHReady, app.EventOptions{
		ProjectID: "proj_123",
		Instance:  "web",
	}))

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected one json line, got %d: %q", len(lines), buf.String())
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &payload); err != nil {
		t.Fatalf("unmarshal event: %v", err)
	}
	if payload["type"] != "event" || payload["name"] != "ssh.ready" {
		t.Fatalf("unexpected event payload: %#v", payload)
	}
}

func TestWriteDocsIndex(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	if err := writeDocsIndex(&buf); err != nil {
		t.Fatalf("writeDocsIndex returned error: %v", err)
	}
	if got := buf.String(); got == "" {
		t.Fatal("expected docs index output")
	}
}
