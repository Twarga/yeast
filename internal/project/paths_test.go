package project

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewPathsScopesRuntimeUnderProjectID(t *testing.T) {
	home := filepath.Join(t.TempDir(), ".yeast")
	metadata := NewMetadata("proj_0123456789abcdef01234567", time.Now())

	paths, err := NewPaths(home, metadata)
	if err != nil {
		t.Fatalf("NewPaths returned error: %v", err)
	}

	wantProjectDir := filepath.Join(home, ProjectsDirName, metadata.ID)
	if paths.ProjectDir != wantProjectDir {
		t.Fatalf("expected project dir %q, got %q", wantProjectDir, paths.ProjectDir)
	}
	if paths.StateFile != filepath.Join(wantProjectDir, StateFileName) {
		t.Fatalf("state file is not scoped under project dir: %q", paths.StateFile)
	}
	if paths.StateLock != filepath.Join(wantProjectDir, StateLockFileName) {
		t.Fatalf("state lock is not scoped under project dir: %q", paths.StateLock)
	}
	if paths.Instances != filepath.Join(wantProjectDir, InstancesDirName) {
		t.Fatalf("instances dir is not scoped under project dir: %q", paths.Instances)
	}
}

func TestInstanceDirIsScopedToProject(t *testing.T) {
	home := filepath.Join(t.TempDir(), ".yeast")
	metadata := NewMetadata("proj_0123456789abcdef01234567", time.Now())
	paths, err := NewPaths(home, metadata)
	if err != nil {
		t.Fatalf("NewPaths returned error: %v", err)
	}

	instanceDir, err := paths.InstanceDir("web")
	if err != nil {
		t.Fatalf("InstanceDir returned error: %v", err)
	}

	want := filepath.Join(home, ProjectsDirName, metadata.ID, InstancesDirName, "web")
	if instanceDir != want {
		t.Fatalf("expected instance dir %q, got %q", want, instanceDir)
	}
	if !strings.HasPrefix(instanceDir, paths.ProjectDir+string(filepath.Separator)) {
		t.Fatalf("instance dir %q escaped project dir %q", instanceDir, paths.ProjectDir)
	}
}

func TestImageCacheIsGlobalSharedPath(t *testing.T) {
	home := filepath.Join(t.TempDir(), ".yeast")
	metadata := NewMetadata("proj_0123456789abcdef01234567", time.Now())
	paths, err := NewPaths(home, metadata)
	if err != nil {
		t.Fatalf("NewPaths returned error: %v", err)
	}

	want := filepath.Join(home, CacheDirName, ImagesDirName)
	if paths.ImageCache != want {
		t.Fatalf("expected global image cache %q, got %q", want, paths.ImageCache)
	}
	if strings.Contains(paths.ImageCache, metadata.ID) {
		t.Fatalf("image cache should be global, got project-scoped path %q", paths.ImageCache)
	}
}

func TestInvalidInstanceNamesRejectedBeforePathCreation(t *testing.T) {
	home := filepath.Join(t.TempDir(), ".yeast")
	metadata := NewMetadata("proj_0123456789abcdef01234567", time.Now())
	paths, err := NewPaths(home, metadata)
	if err != nil {
		t.Fatalf("NewPaths returned error: %v", err)
	}

	for _, name := range []string{"", "../escape", "bad/name", "bad..name", ".hidden"} {
		t.Run(name, func(t *testing.T) {
			if _, err := paths.InstanceDir(name); err == nil {
				t.Fatalf("expected instance name %q to be rejected", name)
			}
		})
	}
}

func TestNewPathsRejectsInvalidMetadata(t *testing.T) {
	home := filepath.Join(t.TempDir(), ".yeast")
	_, err := NewPaths(home, Metadata{
		Schema:    MetadataSchema,
		ID:        "../bad",
		CreatedAt: time.Now(),
	})
	if err == nil {
		t.Fatal("expected invalid metadata error")
	}
}
