package app

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"yeast/internal/project"
)

func TestInitCreatesConfigAndProjectMetadata(t *testing.T) {
	root := t.TempDir()
	service := NewService()

	result, err := service.Init(InitOptions{
		ProjectRoot: root,
		Now:         time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	if !result.Created {
		t.Fatal("expected init result to be created")
	}
	if result.ProjectID == "" {
		t.Fatal("expected project id in init result")
	}
	if _, err := os.Stat(result.ConfigPath); err != nil {
		t.Fatalf("expected config file to exist: %v", err)
	}
	if _, err := os.Stat(result.MetadataPath); err != nil {
		t.Fatalf("expected metadata file to exist: %v", err)
	}

	metadata, err := project.LoadMetadata(root)
	if err != nil {
		t.Fatalf("LoadMetadata returned error: %v", err)
	}
	if metadata.ID != result.ProjectID {
		t.Fatalf("expected metadata id %q, got %q", result.ProjectID, metadata.ID)
	}
}

func TestInitWritesStarterConfig(t *testing.T) {
	root := t.TempDir()
	service := NewService()

	result, err := service.Init(InitOptions{ProjectRoot: root})
	if err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	raw, err := os.ReadFile(result.ConfigPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}
	content := string(raw)
	for _, want := range []string{"version: 1", "name: web", "image: ubuntu-24.04"} {
		if !strings.Contains(content, want) {
			t.Fatalf("expected starter config to contain %q, got:\n%s", want, content)
		}
	}
}

func TestInitFromBuiltinTemplate(t *testing.T) {
	root := t.TempDir()
	service := NewService()

	result, err := service.Init(InitOptions{
		ProjectRoot: root,
		Template:    "caddy-single-vm",
		Now:         time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	if result.Template != "caddy-single-vm" {
		t.Fatalf("expected template in result, got %q", result.Template)
	}
	if result.ProjectID == "" {
		t.Fatal("expected project id")
	}
	for _, file := range []string{"yeast.yaml", "README.md", "site/Caddyfile", "site/index.html"} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(file))); err != nil {
			t.Fatalf("expected template file %s to exist: %v", file, err)
		}
	}
	raw, err := os.ReadFile(filepath.Join(root, "yeast.yaml"))
	if err != nil {
		t.Fatalf("read template config: %v", err)
	}
	if !strings.Contains(string(raw), "caddy") {
		t.Fatalf("expected caddy template config, got:\n%s", string(raw))
	}
	if _, err := project.LoadMetadata(root); err != nil {
		t.Fatalf("expected project metadata: %v", err)
	}
}

func TestInitFromLocalTemplate(t *testing.T) {
	templateDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(templateDir, "template.yaml"), []byte(`name: local-sample
title: Local Sample
description: Local reusable project starter.
category: app
version: "1"
files:
  - yeast.yaml
  - assets/message.txt
`), 0644); err != nil {
		t.Fatalf("write local template metadata: %v", err)
	}
	if err := os.WriteFile(filepath.Join(templateDir, "yeast.yaml"), []byte("version: 1\ninstances:\n  - name: web\n    image: ubuntu-24.04\n"), 0644); err != nil {
		t.Fatalf("write local config: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(templateDir, "assets"), 0755); err != nil {
		t.Fatalf("mkdir local assets: %v", err)
	}
	if err := os.WriteFile(filepath.Join(templateDir, "assets", "message.txt"), []byte("hello\n"), 0644); err != nil {
		t.Fatalf("write local asset: %v", err)
	}

	root := t.TempDir()
	service := NewService()
	result, err := service.Init(InitOptions{ProjectRoot: root, Template: templateDir})
	if err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	if result.Template != "local-sample" {
		t.Fatalf("expected local template in result, got %q", result.Template)
	}
	raw, err := os.ReadFile(filepath.Join(root, "assets", "message.txt"))
	if err != nil {
		t.Fatalf("read local asset: %v", err)
	}
	if string(raw) != "hello\n" {
		t.Fatalf("unexpected local asset: %q", string(raw))
	}
}

func TestInitFromMissingTemplate(t *testing.T) {
	root := t.TempDir()
	service := NewService()

	_, err := service.Init(InitOptions{ProjectRoot: root, Template: "does-not-exist"})
	assertInitAppErrorCode(t, err, ErrorCodeNotFound)
}

func TestInitTemplateConflictMapsToConflict(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("existing\n"), 0644); err != nil {
		t.Fatalf("write existing README: %v", err)
	}
	service := NewService()

	_, err := service.Init(InitOptions{ProjectRoot: root, Template: "ubuntu-basic"})
	assertInitAppErrorCode(t, err, ErrorCodeConflict)
	if !strings.Contains(err.Error(), "template output already exists") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInitFailsClearlyWhenRepeated(t *testing.T) {
	root := t.TempDir()
	service := NewService()

	if _, err := service.Init(InitOptions{ProjectRoot: root}); err != nil {
		t.Fatalf("first Init returned error: %v", err)
	}

	_, err := service.Init(InitOptions{ProjectRoot: root})
	if err == nil {
		t.Fatal("expected repeated init to fail")
	}
	if !errors.Is(err, ErrProjectAlreadyInitialized) {
		t.Fatalf("expected ErrProjectAlreadyInitialized, got %v", err)
	}
	var appErr *AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.Code != ErrorCodeConflict {
		t.Fatalf("expected conflict error code, got %q", appErr.Code)
	}
}

func TestInitClassifiesConfigInspectFailure(t *testing.T) {
	root := t.TempDir()
	blocker := filepath.Join(root, "not-a-directory")
	if err := os.WriteFile(blocker, []byte("x"), 0644); err != nil {
		t.Fatalf("write blocker: %v", err)
	}

	service := NewService()
	_, err := service.Init(InitOptions{ProjectRoot: blocker})
	assertInitAppErrorCode(t, err, ErrorCodeInternal)
	if !strings.Contains(err.Error(), "inspect config file") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInitClassifiesConfigWriteFailure(t *testing.T) {
	root := t.TempDir()
	previousWriteConfigFileAtomic := writeConfigFileAtomic
	defer func() { writeConfigFileAtomic = previousWriteConfigFileAtomic }()
	writeConfigFileAtomic = func(path string, content []byte) error {
		return errors.New("write temp file failed")
	}

	service := NewService()
	_, err := service.Init(InitOptions{ProjectRoot: root})
	assertInitAppErrorCode(t, err, ErrorCodeInternal)
	if !strings.Contains(err.Error(), "write temp file failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func assertInitAppErrorCode(t *testing.T, err error, want ErrorCode) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var appErr *AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.Code != want {
		t.Fatalf("expected error code %q, got %q", want, appErr.Code)
	}
}
