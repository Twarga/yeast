package app

import (
	"testing"
	"time"
	"yeast/internal/state"
)

func TestListInstanceSnapshotsReturnsSortedSnapshots(t *testing.T) {
	currentState := state.New("proj_0123456789abcdef01234567")
	currentState.Instances["web"] = state.InstanceState{
		Snapshots: map[string]state.SnapshotState{
			"late": {
				Name:      "late",
				CreatedAt: time.Date(2026, 5, 18, 16, 0, 0, 0, time.UTC),
			},
			"early": {
				Name:      "early",
				CreatedAt: time.Date(2026, 5, 18, 14, 0, 0, 0, time.UTC),
			},
		},
	}

	got, err := listInstanceSnapshots(currentState, "web")
	if err != nil {
		t.Fatalf("listInstanceSnapshots returned error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 snapshots, got %d", len(got))
	}
	if got[0].Name != "early" || got[1].Name != "late" {
		t.Fatalf("unexpected snapshot order: %#v", got)
	}
}

func TestListInstanceSnapshotsHandlesMissingSnapshotState(t *testing.T) {
	currentState := state.New("proj_0123456789abcdef01234567")
	currentState.Instances["web"] = state.InstanceState{}

	got, err := listInstanceSnapshots(currentState, "web")
	if err != nil {
		t.Fatalf("listInstanceSnapshots returned error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty snapshot list, got %#v", got)
	}
}

func TestListInstanceSnapshotsRequiresTarget(t *testing.T) {
	_, err := listInstanceSnapshots(state.New("proj_0123456789abcdef01234567"), "")
	assertAppErrorCode(t, err, ErrorCodeInvalidArgument)
}

func TestListInstanceSnapshotsRequiresExistingInstance(t *testing.T) {
	_, err := listInstanceSnapshots(state.New("proj_0123456789abcdef01234567"), "web")
	assertAppErrorCode(t, err, ErrorCodeNotFound)
}
