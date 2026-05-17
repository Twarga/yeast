package provision

import (
	"testing"
	"yeast/internal/state"
)

func TestNewResultInitializesProvisioningState(t *testing.T) {
	result := NewResult("/tmp/web/provision.log")

	if result.Status != state.ProvisioningStatusNotStarted {
		t.Fatalf("expected not_started status, got %q", result.Status)
	}
	if result.LogPath != "/tmp/web/provision.log" {
		t.Fatalf("expected log path /tmp/web/provision.log, got %q", result.LogPath)
	}
	if result.Steps == nil {
		t.Fatal("expected steps slice to be initialized")
	}
	if len(result.Steps) != 0 {
		t.Fatalf("expected no initial steps, got %d", len(result.Steps))
	}
}
