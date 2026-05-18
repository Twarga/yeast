package state

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadMissingReturnsEmptyState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")

	state, err := Load(path, "proj_0123456789abcdef01234567")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if state.Schema != Schema {
		t.Fatalf("expected schema %q, got %q", Schema, state.Schema)
	}
	if state.ProjectID != "proj_0123456789abcdef01234567" {
		t.Fatalf("unexpected project id %q", state.ProjectID)
	}
	if state.Instances == nil {
		t.Fatal("expected instances map to be initialized")
	}
}

func TestLoadValidState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	original := New("proj_0123456789abcdef01234567")
	original.Instances["web"] = InstanceState{
		Status: "running",
		PID:    42,
		Snapshots: map[string]SnapshotState{
			"clean": {
				Name:      "clean",
				CreatedAt: time.Date(2026, 5, 18, 14, 30, 0, 0, time.UTC),
				DiskPath:  "/tmp/web/snapshots/clean.qcow2",
			},
		},
	}

	if err := Save(path, original); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	loaded, err := Load(path, original.ProjectID)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if loaded.ProjectID != original.ProjectID {
		t.Fatalf("expected project id %q, got %q", original.ProjectID, loaded.ProjectID)
	}
	if loaded.Instances["web"].PID != 42 {
		t.Fatalf("expected pid 42, got %d", loaded.Instances["web"].PID)
	}
	if loaded.Instances["web"].Snapshots["clean"].DiskPath != "/tmp/web/snapshots/clean.qcow2" {
		t.Fatalf("expected snapshot disk path to round-trip, got %#v", loaded.Instances["web"].Snapshots["clean"])
	}
}

func TestLoadCorruptStateReturnsClearError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(path, []byte("{bad json"), 0644); err != nil {
		t.Fatalf("failed to write corrupt state: %v", err)
	}

	_, err := Load(path, "proj_0123456789abcdef01234567")
	if err == nil {
		t.Fatal("expected corrupt state error")
	}
	if !strings.Contains(err.Error(), "parse state file") {
		t.Fatalf("expected parse state file error, got %v", err)
	}
}

func TestLoadProjectMismatchReturnsClearError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	original := New("proj_aaaaaaaaaaaaaaaaaaaaaaaa")
	if err := Save(path, original); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	_, err := Load(path, "proj_bbbbbbbbbbbbbbbbbbbbbbbb")
	if err == nil {
		t.Fatal("expected project mismatch error")
	}
	if !errors.Is(err, ErrProjectIDMismatch) {
		t.Fatalf("expected ErrProjectIDMismatch, got %v", err)
	}
}

func TestSaveReloadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "state.json")
	original := New("proj_0123456789abcdef01234567")
	original.Instances["web"] = InstanceState{
		Status:       "running",
		PID:          1001,
		ManagementIP: "127.0.0.1",
		SSHPort:      2222,
		RuntimeDir:   "/tmp/web",
		Snapshots: map[string]SnapshotState{
			"baseline": {
				Name:           "baseline",
				CreatedAt:      time.Date(2026, 5, 18, 15, 0, 0, 0, time.UTC),
				Description:    "Initial ready state",
				DiskPath:       "/tmp/web/snapshots/baseline.qcow2",
				SourceDiskSize: "20G",
			},
		},
		ProvisionLogPath:   "/tmp/web/provision.log",
		ProvisioningStatus: ProvisioningStatusRunning,
		LastError:          "none",
	}

	if err := Save(path, original); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	loaded, err := Load(path, original.ProjectID)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if loaded.Instances["web"].RuntimeDir != "/tmp/web" {
		t.Fatalf("expected runtime dir /tmp/web, got %q", loaded.Instances["web"].RuntimeDir)
	}
	if loaded.Instances["web"].ProvisionLogPath != "/tmp/web/provision.log" {
		t.Fatalf("expected provision log path /tmp/web/provision.log, got %q", loaded.Instances["web"].ProvisionLogPath)
	}
	if loaded.Instances["web"].ProvisioningStatus != ProvisioningStatusRunning {
		t.Fatalf("expected provisioning status running, got %q", loaded.Instances["web"].ProvisioningStatus)
	}
	if loaded.Instances["web"].Snapshots["baseline"].Description != "Initial ready state" {
		t.Fatalf("expected snapshot description to round-trip, got %#v", loaded.Instances["web"].Snapshots["baseline"])
	}
}
