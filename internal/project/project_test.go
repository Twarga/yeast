package project

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
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

func TestRemoveLocalProjectFilesRemovesOnlyYeastOwnedFiles(t *testing.T) {
	root := t.TempDir()

	if err := os.MkdirAll(filepath.Join(root, ".yeast"), 0755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".yeast", "project.json"), []byte("{}\n"), 0644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "yeast.yaml"), []byte("version: 1\n"), 0644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("keep me\n"), 0644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := RemoveLocalProjectFiles(root)
	if err != nil {
		t.Fatalf("RemoveLocalProjectFiles returned error: %v", err)
	}

	wantRemoved := []string{
		filepath.Join(root, ".yeast"),
		filepath.Join(root, "yeast.yaml"),
	}
	if !reflect.DeepEqual(result.Removed, wantRemoved) {
		t.Fatalf("unexpected removed paths:\n got: %#v\nwant: %#v", result.Removed, wantRemoved)
	}
	if _, err := os.Stat(filepath.Join(root, ".yeast")); !os.IsNotExist(err) {
		t.Fatalf("expected .yeast to be removed, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "yeast.yaml")); !os.IsNotExist(err) {
		t.Fatalf("expected yeast.yaml to be removed, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "notes.txt")); err != nil {
		t.Fatalf("expected unrelated file to remain, stat err=%v", err)
	}
}
