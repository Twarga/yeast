package templates

import (
	"strings"
	"testing"
)

func TestDecodeMetadata(t *testing.T) {
	t.Parallel()

	metadata, err := DecodeMetadata(strings.NewReader(`name: sample
title: Sample
description: Sample project starter.
category: app
version: "1"
files:
  - yeast.yaml
  - site/index.html
`))
	if err != nil {
		t.Fatalf("DecodeMetadata returned error: %v", err)
	}
	if metadata.Name != "sample" {
		t.Fatalf("unexpected name: %q", metadata.Name)
	}
	if len(metadata.Files) != 2 {
		t.Fatalf("expected 2 files, got %#v", metadata.Files)
	}
}

func TestDecodeMetadataRejectsCorruptYAML(t *testing.T) {
	t.Parallel()

	_, err := DecodeMetadata(strings.NewReader("name: ["))
	if err == nil {
		t.Fatal("expected corrupt metadata to fail")
	}
	if !strings.Contains(err.Error(), "parse template metadata") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateMetadataRequiresFields(t *testing.T) {
	t.Parallel()

	err := ValidateMetadata(Metadata{})
	if err == nil {
		t.Fatal("expected empty metadata to fail")
	}
	if !strings.Contains(err.Error(), "template name") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateMetadataRejectsUnsafeFiles(t *testing.T) {
	t.Parallel()

	metadata := Metadata{
		Name:        "sample",
		Title:       "Sample",
		Description: "Sample project starter.",
		Category:    "app",
		Version:     "1",
		Files:       []string{"yeast.yaml", "../outside"},
	}

	err := ValidateMetadata(metadata)
	if err == nil {
		t.Fatal("expected unsafe file to fail")
	}
	if !strings.Contains(err.Error(), "path must stay inside the template") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateMetadataRejectsDuplicateFiles(t *testing.T) {
	t.Parallel()

	metadata := Metadata{
		Name:        "sample",
		Title:       "Sample",
		Description: "Sample project starter.",
		Category:    "app",
		Version:     "1",
		Files:       []string{"./yeast.yaml", "yeast.yaml"},
	}

	err := ValidateMetadata(metadata)
	if err == nil {
		t.Fatal("expected duplicate file to fail")
	}
	if !strings.Contains(err.Error(), "listed more than once") {
		t.Fatalf("unexpected error: %v", err)
	}
}
