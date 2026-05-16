package state

import "testing"

func TestReconcileDeadPIDBecomesStopped(t *testing.T) {
	state := New("proj_0123456789abcdef01234567")
	state.Instances["web"] = InstanceState{
		Status:       "running",
		PID:          1234,
		ManagementIP: "127.0.0.1",
		SSHPort:      2222,
		LastError:    "",
	}

	changed := Reconcile(&state, ReconcileOptions{
		IsProcessAlive: func(pid int) bool { return false },
	})
	if !changed {
		t.Fatal("expected reconcile to report a change")
	}

	instance := state.Instances["web"]
	if instance.Status != "stopped" {
		t.Fatalf("expected stopped status, got %q", instance.Status)
	}
	if instance.PID != 0 {
		t.Fatalf("expected pid to be cleared, got %d", instance.PID)
	}
	if instance.ManagementIP != "" {
		t.Fatalf("expected management ip to be cleared, got %q", instance.ManagementIP)
	}
	if instance.SSHPort != 0 {
		t.Fatalf("expected ssh port to be cleared, got %d", instance.SSHPort)
	}
	if instance.LastError != "process not running" {
		t.Fatalf("expected last error to be set, got %q", instance.LastError)
	}
}

func TestReconcileStoppedInstanceUnchanged(t *testing.T) {
	state := New("proj_0123456789abcdef01234567")
	state.Instances["web"] = InstanceState{
		Status:    "stopped",
		PID:       0,
		LastError: "",
	}

	changed := Reconcile(&state, ReconcileOptions{
		IsProcessAlive: func(pid int) bool { return false },
	})
	if changed {
		t.Fatal("expected no changes for stopped instance")
	}
	if state.Instances["web"].Status != "stopped" {
		t.Fatalf("expected status to remain stopped, got %q", state.Instances["web"].Status)
	}
}

func TestReconcileRunningAliveInstanceUnchanged(t *testing.T) {
	state := New("proj_0123456789abcdef01234567")
	state.Instances["web"] = InstanceState{
		Status:       "running",
		PID:          1234,
		ManagementIP: "127.0.0.1",
		SSHPort:      2222,
	}

	changed := Reconcile(&state, ReconcileOptions{
		IsProcessAlive: func(pid int) bool { return true },
	})
	if changed {
		t.Fatal("expected no changes for live running instance")
	}

	instance := state.Instances["web"]
	if instance.Status != "running" {
		t.Fatalf("expected status to remain running, got %q", instance.Status)
	}
	if instance.PID != 1234 {
		t.Fatalf("expected pid to remain 1234, got %d", instance.PID)
	}
}
