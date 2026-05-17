package app

import (
	"errors"
	"fmt"
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

	_, err := service.Pull(PullOptions{ImageName: "debian-13"})
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
