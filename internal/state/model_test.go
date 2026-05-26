package state

import (
	"encoding/json"
	"testing"
	"time"
)

func TestStateJSONRoundTrip(t *testing.T) {
	original := New("proj_0123456789abcdef01234567")
	original.Instances["web"] = InstanceState{
		Status:       "running",
		PID:          1234,
		ManagementIP: "127.0.0.1",
		SSHPort:      2222,
		User:         "yeast",
		RuntimeDir:   "/home/twarga/.yeast/projects/proj_0123456789abcdef01234567/instances/web",
		Snapshots: map[string]SnapshotState{
			"clean": {
				Name:           "clean",
				CreatedAt:      time.Date(2026, 5, 18, 14, 0, 0, 0, time.UTC),
				Description:    "Clean post-provision baseline",
				DiskPath:       "/home/twarga/.yeast/projects/proj_0123456789abcdef01234567/instances/web/snapshots/clean.qcow2",
				SourceDiskSize: "20G",
			},
		},
		ProvisionLogPath:   "/home/twarga/.yeast/projects/proj_0123456789abcdef01234567/instances/web/provision.log",
		ProvisioningStatus: ProvisioningStatusReady,
		LastError:          "",
	}

	raw, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}

	var decoded State
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}

	if decoded.Schema != Schema {
		t.Fatalf("expected schema %q, got %q", Schema, decoded.Schema)
	}
	if decoded.ProjectID != original.ProjectID {
		t.Fatalf("expected project id %q, got %q", original.ProjectID, decoded.ProjectID)
	}
	if len(decoded.Instances) != 1 {
		t.Fatalf("expected 1 instance, got %d", len(decoded.Instances))
	}

	instance, ok := decoded.Instances["web"]
	if !ok {
		t.Fatal("expected instance state for web")
	}
	if instance.Status != "running" {
		t.Fatalf("expected status running, got %q", instance.Status)
	}
	if instance.PID != 1234 {
		t.Fatalf("expected pid 1234, got %d", instance.PID)
	}
	if instance.ManagementIP != "127.0.0.1" {
		t.Fatalf("expected management ip 127.0.0.1, got %q", instance.ManagementIP)
	}
	if instance.SSHPort != 2222 {
		t.Fatalf("expected ssh port 2222, got %d", instance.SSHPort)
	}
	if instance.User != "yeast" {
		t.Fatalf("expected user yeast, got %q", instance.User)
	}
	if instance.RuntimeDir == "" {
		t.Fatal("expected runtime dir to survive round trip")
	}
	snapshot, ok := instance.Snapshots["clean"]
	if !ok {
		t.Fatal("expected snapshot metadata to survive round trip")
	}
	if snapshot.Name != "clean" {
		t.Fatalf("expected snapshot name clean, got %q", snapshot.Name)
	}
	if snapshot.Description != "Clean post-provision baseline" {
		t.Fatalf("unexpected snapshot description %q", snapshot.Description)
	}
	if snapshot.DiskPath == "" {
		t.Fatal("expected snapshot disk path to survive round trip")
	}
	if snapshot.SourceDiskSize != "20G" {
		t.Fatalf("expected snapshot source disk size 20G, got %q", snapshot.SourceDiskSize)
	}
	if instance.ProvisionLogPath == "" {
		t.Fatal("expected provision log path to survive round trip")
	}
	if instance.ProvisioningStatus != ProvisioningStatusReady {
		t.Fatalf("expected provisioning status provisioned, got %q", instance.ProvisioningStatus)
	}
}

func TestNewInitializesVersionedState(t *testing.T) {
	state := New("proj_0123456789abcdef01234567")

	if state.Schema != Schema {
		t.Fatalf("expected schema %q, got %q", Schema, state.Schema)
	}
	if state.ProjectID != "proj_0123456789abcdef01234567" {
		t.Fatalf("unexpected project id %q", state.ProjectID)
	}
	if state.Instances == nil {
		t.Fatal("expected instances map to be initialized")
	}
	if len(state.Instances) != 0 {
		t.Fatalf("expected empty instances map, got %d entries", len(state.Instances))
	}
}

func TestProvisioningStatusConstants(t *testing.T) {
	if ProvisioningStatusNotStarted != "not_started" {
		t.Fatalf("unexpected not_started value %q", ProvisioningStatusNotStarted)
	}
	if ProvisioningStatusRunning != "running" {
		t.Fatalf("unexpected running value %q", ProvisioningStatusRunning)
	}
	if ProvisioningStatusReady != "provisioned" {
		t.Fatalf("unexpected provisioned value %q", ProvisioningStatusReady)
	}
	if ProvisioningStatusFailed != "failed" {
		t.Fatalf("unexpected failed value %q", ProvisioningStatusFailed)
	}
}

func TestSnapshotStateFields(t *testing.T) {
	snapshot := SnapshotState{
		Name:           "baseline",
		CreatedAt:      time.Date(2026, 5, 18, 15, 0, 0, 0, time.UTC),
		Description:    "Provisioned baseline",
		DiskPath:       "/tmp/baseline.qcow2",
		SourceDiskSize: "20G",
	}

	if snapshot.Name != "baseline" {
		t.Fatalf("unexpected snapshot name %q", snapshot.Name)
	}
	if snapshot.DiskPath != "/tmp/baseline.qcow2" {
		t.Fatalf("unexpected disk path %q", snapshot.DiskPath)
	}
	if snapshot.SourceDiskSize != "20G" {
		t.Fatalf("unexpected source disk size %q", snapshot.SourceDiskSize)
	}
}
