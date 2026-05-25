package templates

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestMaterializeBuiltinTemplate(t *testing.T) {
	t.Parallel()

	template, ok, err := LookupBuiltin("caddy-single-vm")
	if err != nil {
		t.Fatalf("LookupBuiltin returned error: %v", err)
	}
	if !ok {
		t.Fatal("expected built-in template")
	}

	destination := t.TempDir()
	result, err := Materialize(template, MaterializeOptions{Destination: destination})
	if err != nil {
		t.Fatalf("Materialize returned error: %v", err)
	}

	wantFiles := []string{"yeast.yaml", "README.md", "site/Caddyfile", "site/index.html"}
	if !reflect.DeepEqual(result.Files, wantFiles) {
		t.Fatalf("unexpected materialized files:\n got: %#v\nwant: %#v", result.Files, wantFiles)
	}
	for _, file := range wantFiles {
		if _, err := os.Stat(filepath.Join(destination, filepath.FromSlash(file))); err != nil {
			t.Fatalf("expected %s to exist: %v", file, err)
		}
	}
	raw, err := os.ReadFile(filepath.Join(destination, "yeast.yaml"))
	if err != nil {
		t.Fatalf("read yeast.yaml: %v", err)
	}
	if !strings.Contains(string(raw), "caddy") {
		t.Fatalf("expected caddy config, got:\n%s", string(raw))
	}
}

func TestMaterializeLocalTemplate(t *testing.T) {
	t.Parallel()

	source := t.TempDir()
	writeLocalMetadata(t, source, `name: local-sample
title: Local Sample
description: Local reusable project starter.
category: app
version: "1"
files:
  - yeast.yaml
  - files/app.txt
`)
	if err := os.WriteFile(filepath.Join(source, "yeast.yaml"), []byte("version: 1\ninstances:\n  - name: web\n    image: ubuntu-24.04\n"), 0644); err != nil {
		t.Fatalf("write local config: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(source, "files"), 0755); err != nil {
		t.Fatalf("mkdir local files: %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "files", "app.txt"), []byte("hello\n"), 0644); err != nil {
		t.Fatalf("write local asset: %v", err)
	}

	template, err := LoadLocal(source)
	if err != nil {
		t.Fatalf("LoadLocal returned error: %v", err)
	}
	destination := t.TempDir()
	if _, err := Materialize(template, MaterializeOptions{Destination: destination}); err != nil {
		t.Fatalf("Materialize returned error: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(destination, "files", "app.txt"))
	if err != nil {
		t.Fatalf("read copied asset: %v", err)
	}
	if string(raw) != "hello\n" {
		t.Fatalf("unexpected copied asset: %q", string(raw))
	}
}

func TestMaterializeRejectsExistingOutput(t *testing.T) {
	t.Parallel()

	template, ok, err := LookupBuiltin("ubuntu-basic")
	if err != nil {
		t.Fatalf("LookupBuiltin returned error: %v", err)
	}
	if !ok {
		t.Fatal("expected built-in template")
	}
	destination := t.TempDir()
	if err := os.WriteFile(filepath.Join(destination, "yeast.yaml"), []byte("existing\n"), 0644); err != nil {
		t.Fatalf("write existing file: %v", err)
	}

	_, err = Materialize(template, MaterializeOptions{Destination: destination})
	if err == nil {
		t.Fatal("expected existing output to fail")
	}
	if !errors.Is(err, ErrOutputExists) {
		t.Fatalf("expected ErrOutputExists, got %v", err)
	}
}

func TestMaterializeRejectsMissingLocalFile(t *testing.T) {
	t.Parallel()

	source := t.TempDir()
	writeLocalMetadata(t, source, `name: local-missing
title: Local Missing
description: Local template with a missing file.
category: app
version: "1"
files:
  - yeast.yaml
`)
	template, err := LoadLocal(source)
	if err != nil {
		t.Fatalf("LoadLocal returned error: %v", err)
	}

	_, err = Materialize(template, MaterializeOptions{Destination: t.TempDir()})
	if err == nil {
		t.Fatal("expected missing local file to fail")
	}
	if !strings.Contains(err.Error(), "read local template file") {
		t.Fatalf("unexpected error: %v", err)
	}
}
