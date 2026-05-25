package templates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadLocal(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeLocalMetadata(t, dir, `name: local-sample
title: Local Sample
description: Local reusable project starter.
category: lab
version: "1"
files:
  - yeast.yaml
`)

	template, err := LoadLocal(dir)
	if err != nil {
		t.Fatalf("LoadLocal returned error: %v", err)
	}
	if template.Source != SourceLocal {
		t.Fatalf("expected local source, got %q", template.Source)
	}
	if template.Path != filepath.Clean(dir) {
		t.Fatalf("unexpected path: %q", template.Path)
	}
	if template.Metadata.Name != "local-sample" {
		t.Fatalf("unexpected template name: %q", template.Metadata.Name)
	}
}

func TestLoadLocalRejectsMissingMetadata(t *testing.T) {
	t.Parallel()

	_, err := LoadLocal(t.TempDir())
	if err == nil {
		t.Fatal("expected missing metadata to fail")
	}
	if !strings.Contains(err.Error(), "read template metadata") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadLocalRejectsCorruptMetadata(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeLocalMetadata(t, dir, "name: [")

	_, err := LoadLocal(dir)
	if err == nil {
		t.Fatal("expected corrupt metadata to fail")
	}
	if !strings.Contains(err.Error(), "load template metadata") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadLocalRejectsFilePath(t *testing.T) {
	t.Parallel()

	file := filepath.Join(t.TempDir(), "template.yaml")
	if err := os.WriteFile(file, []byte("name: file\n"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err := LoadLocal(file)
	if err == nil {
		t.Fatal("expected file path to fail")
	}
	if !strings.Contains(err.Error(), "is not a directory") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func writeLocalMetadata(t *testing.T, dir, content string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(dir, MetadataFileName), []byte(content), 0644); err != nil {
		t.Fatalf("write local metadata: %v", err)
	}
}
