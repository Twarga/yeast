package images

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveCachePathsUnderCacheRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".yeast", "cache", "images")

	paths, err := ResolveCachePaths(root, "ubuntu-24.04")
	if err != nil {
		t.Fatalf("ResolveCachePaths returned error: %v", err)
	}

	wantDir := filepath.Join(root, "ubuntu-24.04")
	if paths.ImageDir != wantDir {
		t.Fatalf("expected image dir %q, got %q", wantDir, paths.ImageDir)
	}
	if paths.ImageFile != filepath.Join(wantDir, ImageFileName) {
		t.Fatalf("expected image file under image dir, got %q", paths.ImageFile)
	}
	if paths.ManifestFile != filepath.Join(wantDir, ManifestFileName) {
		t.Fatalf("expected manifest file under image dir, got %q", paths.ManifestFile)
	}
}

func TestImageNameSafe(t *testing.T) {
	valid := []string{"ubuntu-22.04", "ubuntu-24.04", "kali-2026.1"}
	for _, name := range valid {
		if !IsSafeImageName(name) {
			t.Fatalf("expected %q to be valid", name)
		}
	}

	invalid := []string{"", "../escape", "bad/name", "bad name", ".hidden", "UPPER", "bad..name"}
	for _, name := range invalid {
		if IsSafeImageName(name) {
			t.Fatalf("expected %q to be invalid", name)
		}
		if _, err := ResolveCachePaths("/tmp/cache/images", name); err == nil {
			t.Fatalf("expected ResolveCachePaths to reject %q", name)
		}
	}
}

func TestResolveCachePathsKeepsFilesUnderCacheRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".yeast", "cache", "images")

	paths, err := ResolveCachePaths(root, "ubuntu-24.04")
	if err != nil {
		t.Fatalf("ResolveCachePaths returned error: %v", err)
	}

	prefix := root + string(filepath.Separator)
	if !strings.HasPrefix(paths.ImageDir, prefix) {
		t.Fatalf("expected image dir %q under cache root %q", paths.ImageDir, root)
	}
	if !strings.HasPrefix(paths.ImageFile, prefix) {
		t.Fatalf("expected image file %q under cache root %q", paths.ImageFile, root)
	}
	if !strings.HasPrefix(paths.ManifestFile, prefix) {
		t.Fatalf("expected manifest file %q under cache root %q", paths.ManifestFile, root)
	}
}
