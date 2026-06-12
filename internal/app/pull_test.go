package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"yeast/internal/images"
)

func TestPullListReturnsSupportedImages(t *testing.T) {
	service := NewService()

	result, err := service.Pull(PullOptions{List: true})
	if err != nil {
		t.Fatalf("Pull returned error: %v", err)
	}
	if !result.List {
		t.Fatal("expected list result")
	}
	if len(result.Images) == 0 {
		t.Fatal("expected supported images")
	}
}

func TestPullUnsupportedImageFailsClearly(t *testing.T) {
	service := NewService()

	_, err := service.Pull(PullOptions{ImageName: "nonexistent-distro-99"})
	if err == nil {
		t.Fatal("expected unsupported image error")
	}
	if !errors.Is(err, ErrUnsupportedImage) {
		t.Fatalf("expected ErrUnsupportedImage, got %v", err)
	}
	var appErr *AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.Code != ErrorCodeInvalidArgument {
		t.Fatalf("expected invalid_argument error code, got %q", appErr.Code)
	}
}

func TestPullDownloadsKnownImage(t *testing.T) {
	service := NewService()
	tempHome := t.TempDir()
	var gotImage images.TrustedImage
	var gotDestination string

	service.resolveYeastHome = func() (string, error) {
		return filepath.Join(tempHome, ".yeast"), nil
	}
	service.downloadImage = func(image images.TrustedImage, destination string, options images.DownloadOptions) error {
		gotImage = image
		gotDestination = destination
		return nil
	}

	result, err := service.Pull(PullOptions{ImageName: "ubuntu-24.04"})
	if err != nil {
		t.Fatalf("Pull returned error: %v", err)
	}
	if gotImage.Name != "ubuntu-24.04" {
		t.Fatalf("expected ubuntu-24.04 image, got %q", gotImage.Name)
	}
	wantDestination := filepath.Join(tempHome, ".yeast", "cache", "images", "ubuntu-24.04", images.ImageFileName)
	if gotDestination != wantDestination {
		t.Fatalf("expected destination %q, got %q", wantDestination, gotDestination)
	}
	if result.ImagePath != wantDestination {
		t.Fatalf("expected result image path %q, got %q", wantDestination, result.ImagePath)
	}
}

func TestPullManualImageShowsHint(t *testing.T) {
	service := NewService()

	result, err := service.Pull(PullOptions{ImageName: "kali-2026.1"})
	if err != nil {
		t.Fatalf("Pull returned error: %v", err)
	}
	if result.ImageName != "kali-2026.1" {
		t.Fatalf("expected kali-2026.1, got %q", result.ImageName)
	}
	if result.ManualHint == "" {
		t.Fatal("expected manual hint for kali image")
	}
	if result.ImagePath != "" {
		t.Fatal("expected no image path for manual image")
	}
}

func TestPullListReturnsCategorizedImages(t *testing.T) {
	service := NewService()

	result, err := service.Pull(PullOptions{List: true})
	if err != nil {
		t.Fatalf("Pull returned error: %v", err)
	}
	if !result.List {
		t.Fatal("expected list result")
	}
	if len(result.Images) == 0 {
		t.Fatal("expected supported images")
	}
	if len(result.ImageGroups) == 0 {
		t.Fatal("expected categorized image groups")
	}

	// Verify categories exist.
	categories := make(map[string]bool)
	for _, group := range result.ImageGroups {
		categories[group.Category] = true
		if len(group.Images) == 0 {
			t.Fatalf("category %q has no images", group.Category)
		}
	}
	for _, want := range []string{"general", "devops", "enterprise", "security", "minimal", "niche"} {
		if !categories[want] {
			t.Fatalf("expected category %q", want)
		}
	}
}

func TestPullCachedReturnsCachedImages(t *testing.T) {
	service := NewService()
	tempHome := t.TempDir()

	service.resolveYeastHome = func() (string, error) {
		return filepath.Join(tempHome, ".yeast"), nil
	}

	// Create a fake cached image.
	imgDir := filepath.Join(tempHome, ".yeast", "cache", "images", "ubuntu-24.04")
	if err := os.MkdirAll(imgDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(imgDir, images.ImageFileName), []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := service.Pull(PullOptions{Cached: true})
	if err != nil {
		t.Fatalf("Pull returned error: %v", err)
	}
	if len(result.CachedImages) != 1 {
		t.Fatalf("expected 1 cached image, got %d", len(result.CachedImages))
	}
	if result.CachedImages[0].Name != "ubuntu-24.04" {
		t.Fatalf("expected ubuntu-24.04, got %q", result.CachedImages[0].Name)
	}
}

func TestPullClassifiesYeastHomeResolutionFailure(t *testing.T) {
	service := NewService()
	service.resolveYeastHome = func() (string, error) {
		return "", errors.New("home lookup failed")
	}

	_, err := service.Pull(PullOptions{ImageName: "ubuntu-24.04"})
	assertAppErrorCode(t, err, ErrorCodeInternal)
}

func TestPullClassifiesCachePathFailure(t *testing.T) {
	service := NewService()
	service.resolveYeastHome = func() (string, error) {
		return "", nil
	}

	_, err := service.Pull(PullOptions{ImageName: "ubuntu-24.04"})
	assertAppErrorCode(t, err, ErrorCodeInternal)
}

func TestPullClassifiesDownloadFailure(t *testing.T) {
	service := NewService()
	tempHome := t.TempDir()
	service.resolveYeastHome = func() (string, error) {
		return filepath.Join(tempHome, ".yeast"), nil
	}
	service.downloadImage = func(image images.TrustedImage, destination string, options images.DownloadOptions) error {
		return fmt.Errorf("download failed")
	}

	_, err := service.Pull(PullOptions{ImageName: "ubuntu-24.04"})
	assertAppErrorCode(t, err, ErrorCodeInternal)
}
