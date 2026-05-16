package app

import (
	"errors"
	"os"
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
}
