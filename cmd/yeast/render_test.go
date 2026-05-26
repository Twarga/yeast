package main

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
	"yeast/internal/app"
	"yeast/internal/state"
)

func TestRenderCommandOutputJSONForCoreCommands(t *testing.T) {
	tests := []struct {
		name    string
		command string
		data    any
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
		},
		{
			name:    "pull",
			command: "pull",
			data: app.PullResult{
				ImageName: "ubuntu-24.04",
				ImagePath: "/tmp/cache/images/ubuntu-24.04/image.qcow2",
			},
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
		},
		{
			name:    "status",
			command: "status",
			data: app.StatusResult{
				ProjectID: "proj_123",
				Instances: []app.StatusInstanceResult{
					{Name: "web", Status: "running", SSHPort: 2222, LabIP: "10.10.10.10"},
				},
			},
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
					LabIP:              "10.10.10.10",
					RuntimeDir:         "/tmp/web",
					ProvisionLogPath:   "/tmp/web/provision.log",
					ProvisioningStatus: state.ProvisioningStatusReady,
				},
				SnapshotNames: []string{"clean"},
				SnapshotCount: 1,
			},
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
		},
		{
			name:    "delete-snapshot",
			command: "delete-snapshot",
			data: app.DeleteSnapshotResult{
				ProjectID: "proj_123",
				Instance:  "web",
				Snapshot:  "clean",
			},
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
			if _, ok := payload["data"]; !ok {
				t.Fatalf("expected data field, payload=%#v", payload)
			}
		})
	}
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
