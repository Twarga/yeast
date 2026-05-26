package app

import (
	"encoding/json"
	"testing"
	"time"
)

func TestEventNameValuesAreStable(t *testing.T) {
	t.Parallel()

	tests := map[EventName]string{
		EventProjectLoaded:      "project.loaded",
		EventConfigValidated:    "config.validated",
		EventImageReady:         "image.ready",
		EventDiskReady:          "disk.ready",
		EventCloudInitGenerated: "cloud_init.generated",
		EventVMStarting:         "vm.starting",
		EventSSHWaiting:         "ssh.waiting",
		EventSSHReady:           "ssh.ready",
		EventProvisionStarted:   "provision.started",
		EventProvisionFinished:  "provision.finished",
		EventSnapshotCreated:    "snapshot.created",
		EventRestoreStarted:     "restore.started",
		EventRestoreFinished:    "restore.finished",
		EventInstanceReady:      "instance.ready",
		EventInstanceStopped:    "instance.stopped",
		EventInstanceDestroyed:  "instance.destroyed",
		EventWorkflowCompleted:  "workflow.completed",
		EventWorkflowFailed:     "workflow.failed",
	}

	for name, want := range tests {
		if string(name) != want {
			t.Fatalf("unexpected event name: got %q want %q", name, want)
		}
	}
}

func TestNewEventBuildsStableEnvelope(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 5, 26, 13, 0, 0, 0, time.UTC)
	event := NewEvent("up", EventSSHReady, EventOptions{
		ProjectID: "proj_123",
		Instance:  "web",
		Message:   "SSH is ready",
		Now:       now,
		Data: map[string]any{
			"ssh_port": 2222,
		},
	})

	if event.SchemaVersion != "yeast.v1" {
		t.Fatalf("unexpected schema version: %q", event.SchemaVersion)
	}
	if event.Type != "event" {
		t.Fatalf("unexpected event type: %q", event.Type)
	}
	if event.Name != EventSSHReady {
		t.Fatalf("unexpected event name: %q", event.Name)
	}
	if event.Command != "up" {
		t.Fatalf("unexpected command: %q", event.Command)
	}
	if event.ProjectID != "proj_123" || event.Instance != "web" {
		t.Fatalf("unexpected target fields: %#v", event)
	}

	raw, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("unmarshal event: %v", err)
	}
	if payload["schema_version"] != "yeast.v1" {
		t.Fatalf("expected schema_version in JSON, got %#v", payload)
	}
	if payload["type"] != "event" {
		t.Fatalf("expected type=event in JSON, got %#v", payload)
	}
	if payload["name"] != "ssh.ready" {
		t.Fatalf("expected name ssh.ready in JSON, got %#v", payload)
	}
}
