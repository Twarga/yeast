package state

import (
	"testing"
	"time"
)

func TestSortedSnapshotsReturnsStableCreatedAtOrder(t *testing.T) {
	instance := InstanceState{
		Snapshots: map[string]SnapshotState{
			"late": {
				Name:      "late",
				CreatedAt: time.Date(2026, 5, 18, 16, 0, 0, 0, time.UTC),
			},
			"early": {
				Name:      "early",
				CreatedAt: time.Date(2026, 5, 18, 14, 0, 0, 0, time.UTC),
			},
			"middle": {
				Name:      "middle",
				CreatedAt: time.Date(2026, 5, 18, 15, 0, 0, 0, time.UTC),
			},
		},
	}

	got := SortedSnapshots(instance)
	if len(got) != 3 {
		t.Fatalf("expected 3 snapshots, got %d", len(got))
	}
	if got[0].Name != "early" || got[1].Name != "middle" || got[2].Name != "late" {
		t.Fatalf("unexpected snapshot order: %#v", got)
	}
}

func TestSortedSnapshotsBreaksTimestampTiesByName(t *testing.T) {
	createdAt := time.Date(2026, 5, 18, 14, 0, 0, 0, time.UTC)
	instance := InstanceState{
		Snapshots: map[string]SnapshotState{
			"b": {Name: "b", CreatedAt: createdAt},
			"a": {Name: "a", CreatedAt: createdAt},
		},
	}

	got := SortedSnapshots(instance)
	if len(got) != 2 {
		t.Fatalf("expected 2 snapshots, got %d", len(got))
	}
	if got[0].Name != "a" || got[1].Name != "b" {
		t.Fatalf("unexpected tied snapshot order: %#v", got)
	}
}

func TestSortedSnapshotsHandlesMissingSnapshotMap(t *testing.T) {
	got := SortedSnapshots(InstanceState{})
	if got != nil {
		t.Fatalf("expected nil snapshot list, got %#v", got)
	}
}
