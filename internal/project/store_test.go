package project

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestEnsureMetadataCreatesMetadataInProject(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC)

	metadata, err := EnsureMetadata(root, now)
	if err != nil {
		t.Fatalf("EnsureMetadata returned error: %v", err)
	}

	if metadata.Schema != MetadataSchema {
		t.Fatalf("expected schema %q, got %q", MetadataSchema, metadata.Schema)
	}
	if !IsValidID(metadata.ID) {
		t.Fatalf("expected valid generated id, got %q", metadata.ID)
	}
	if got := metadata.CreatedAt.Format(time.RFC3339); got != "2026-05-16T12:00:00Z" {
		t.Fatalf("expected created_at 2026-05-16T12:00:00Z, got %s", got)
	}
	if _, err := os.Stat(MetadataPath(root)); err != nil {
		t.Fatalf("expected metadata file to exist: %v", err)
	}
}

func TestEnsureMetadataLoadsSameIDOnSecondCall(t *testing.T) {
	root := t.TempDir()
	firstTime := time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC)
	secondTime := time.Date(2027, 1, 1, 12, 0, 0, 0, time.UTC)

	first, err := EnsureMetadata(root, firstTime)
	if err != nil {
		t.Fatalf("first EnsureMetadata returned error: %v", err)
	}
	second, err := EnsureMetadata(root, secondTime)
	if err != nil {
		t.Fatalf("second EnsureMetadata returned error: %v", err)
	}

	if second.ID != first.ID {
		t.Fatalf("expected stable project id %q, got %q", first.ID, second.ID)
	}
	if !second.CreatedAt.Equal(first.CreatedAt) {
		t.Fatalf("expected stable created_at %s, got %s", first.CreatedAt, second.CreatedAt)
	}
}

func TestLoadMetadataReturnsClearErrorForCorruptJSON(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, MetadataDirName), 0755); err != nil {
		t.Fatalf("failed to create metadata dir: %v", err)
	}
	if err := os.WriteFile(MetadataPath(root), []byte("{bad json"), 0644); err != nil {
		t.Fatalf("failed to write corrupt metadata: %v", err)
	}

	_, err := LoadMetadata(root)
	if err == nil {
		t.Fatal("expected corrupt metadata error")
	}
	if !strings.Contains(err.Error(), "parse project metadata") {
		t.Fatalf("expected parse project metadata error, got %v", err)
	}
	if !strings.Contains(err.Error(), MetadataPath(root)) {
		t.Fatalf("expected error to include metadata path, got %v", err)
	}
}

func TestLoadMetadataReturnsNotFoundForMissingMetadata(t *testing.T) {
	root := t.TempDir()

	_, err := LoadMetadata(root)
	if err == nil {
		t.Fatal("expected missing metadata error")
	}
	if !errors.Is(err, ErrMetadataNotFound) {
		t.Fatalf("expected ErrMetadataNotFound, got %v", err)
	}
}

func TestLoadMetadataRejectsInvalidMetadata(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, MetadataDirName), 0755); err != nil {
		t.Fatalf("failed to create metadata dir: %v", err)
	}
	raw := []byte(`{"schema":"yeast.project.v1","id":"../bad","created_at":"2026-05-16T12:00:00Z"}`)
	if err := os.WriteFile(MetadataPath(root), raw, 0644); err != nil {
		t.Fatalf("failed to write metadata: %v", err)
	}

	_, err := LoadMetadata(root)
	if err == nil {
		t.Fatal("expected invalid metadata error")
	}
	if !strings.Contains(err.Error(), "invalid project metadata") {
		t.Fatalf("expected invalid project metadata error, got %v", err)
	}
}
