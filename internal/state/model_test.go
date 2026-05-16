package state

import (
	"encoding/json"
	"testing"
)

func TestStateJSONRoundTrip(t *testing.T) {
	original := New("proj_0123456789abcdef01234567")
	original.Instances["web"] = InstanceState{
		Status:             "running",
		PID:                1234,
		ManagementIP:       "127.0.0.1",
		SSHPort:            2222,
		RuntimeDir:         "/home/twarga/.yeast/projects/proj_0123456789abcdef01234567/instances/web",
		ProvisioningStatus: "provisioned",
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
	if instance.RuntimeDir == "" {
		t.Fatal("expected runtime dir to survive round trip")
	}
	if instance.ProvisioningStatus != "provisioned" {
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
