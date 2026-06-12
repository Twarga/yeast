package images

import (
	"os"
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

func TestRemoveCachedImage(t *testing.T) {
	root := t.TempDir()

	// Create a fake cached image.
	imgDir := filepath.Join(root, "ubuntu-24.04")
	if err := os.MkdirAll(imgDir, 0755); err != nil {
		t.Fatal(err)
	}
	data := strings.Repeat("x", 1024)
	if err := os.WriteFile(filepath.Join(imgDir, ImageFileName), []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	size, err := RemoveCachedImage(root, "ubuntu-24.04")
	if err != nil {
		t.Fatalf("RemoveCachedImage returned error: %v", err)
	}
	if size != 1024 {
		t.Fatalf("expected freed size 1024, got %d", size)
	}

	// Verify directory is gone.
	if _, err := os.Stat(imgDir); !os.IsNotExist(err) {
		t.Fatalf("expected image dir to be removed, err=%v", err)
	}

	// Removing again should fail.
	_, err = RemoveCachedImage(root, "ubuntu-24.04")
	if err == nil {
		t.Fatal("expected error removing non-existent image")
	}
}

func TestRemoveAllCached(t *testing.T) {
	root := t.TempDir()

	// Create two cached images.
	for _, name := range []string{"ubuntu-24.04", "debian-12"} {
		imgDir := filepath.Join(root, name)
		if err := os.MkdirAll(imgDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(imgDir, ImageFileName), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	size, err := RemoveAllCached(root)
	if err != nil {
		t.Fatalf("RemoveAllCached returned error: %v", err)
	}
	if size != 2 {
		t.Fatalf("expected freed size 2, got %d", size)
	}

	cached := GetCachedImages(root)
	if len(cached) != 0 {
		t.Fatalf("expected no cached images, got %v", cached)
	}
}
