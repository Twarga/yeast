package project

import (
	"encoding/json"
	"testing"
	"time"
)

func TestMetadataSerialization(t *testing.T) {
	createdAt := time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC)
	metadata := NewMetadata("proj_abc123", createdAt)

	raw, err := json.Marshal(metadata)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}

	want := `{"schema":"yeast.project.v1","id":"proj_abc123","created_at":"2026-05-16T12:00:00Z"}`
	if string(raw) != want {
		t.Fatalf("unexpected JSON:\n got: %s\nwant: %s", string(raw), want)
	}
}

func TestMetadataDeserialization(t *testing.T) {
	raw := []byte(`{"schema":"yeast.project.v1","id":"proj_abc123","created_at":"2026-05-16T12:00:00Z"}`)

	var metadata Metadata
	if err := json.Unmarshal(raw, &metadata); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}

	if metadata.Schema != MetadataSchema {
		t.Fatalf("expected schema %q, got %q", MetadataSchema, metadata.Schema)
	}
	if metadata.ID != "proj_abc123" {
		t.Fatalf("expected id proj_abc123, got %q", metadata.ID)
	}
	if got := metadata.CreatedAt.Format(time.RFC3339); got != "2026-05-16T12:00:00Z" {
		t.Fatalf("expected created_at 2026-05-16T12:00:00Z, got %s", got)
	}
}

func TestNewMetadataStoresUTCTime(t *testing.T) {
	location := time.FixedZone("test", 3600)
	createdAt := time.Date(2026, 5, 16, 13, 0, 0, 0, location)
	metadata := NewMetadata("proj_abc123", createdAt)

	if metadata.CreatedAt.Location() != time.UTC {
		t.Fatalf("expected UTC location, got %s", metadata.CreatedAt.Location())
	}
	if got := metadata.CreatedAt.Format(time.RFC3339); got != "2026-05-16T12:00:00Z" {
		t.Fatalf("expected normalized UTC time, got %s", got)
	}
}

