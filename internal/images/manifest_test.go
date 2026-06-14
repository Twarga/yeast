package images

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestSupportedImagesSorted(t *testing.T) {
	got := SupportedImages()
	want := []string{
		"alma-9",
		"alpine-3.21",
		"amazon-linux-2023",
		"arch-linux",
		"centos-stream-9",
		"debian-12",
		"debian-13",
		"fedora-41",
		"fedora-42",
		"kali-2026.1",
		"nixos-24.11",
		"opensuse-leap-15.6",
		"parrot-security-7.1",
		"rocky-9",
		"ubuntu-22.04",
		"ubuntu-24.04",
	}

	if len(got) != len(want) {
		t.Fatalf("expected %d images, got %d\nwant: %v\ngot:  %v", len(want), len(got), want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected supported image %q at index %d, got %q", want[i], i, got[i])
		}
	}
}

func TestLookupKnownImage(t *testing.T) {
	image, ok := Lookup("ubuntu-24.04")
	if !ok {
		t.Fatal("expected known image lookup to succeed")
	}
	if image.Name != "ubuntu-24.04" {
		t.Fatalf("expected image name ubuntu-24.04, got %q", image.Name)
	}
	if image.URL == "" {
		t.Fatal("expected trusted image URL to be set")
	}
	if image.SHA256 == "" {
		t.Fatal("expected trusted image SHA256 to be set")
	}
	if !image.CloudInit {
		t.Fatal("expected ubuntu-24.04 to support cloud-init")
	}
	if image.Category != CategoryGeneral {
		t.Fatalf("expected category general, got %q", image.Category)
	}
}

func TestLookupUnknownImage(t *testing.T) {
	if _, ok := Lookup("nonexistent-99"); ok {
		t.Fatal("expected unknown image lookup to fail")
	}
}

func TestLookupManualImage(t *testing.T) {
	image, ok := Lookup("kali-2026.1")
	if !ok {
		t.Fatal("expected kali-2026.1 lookup to succeed")
	}
	if image.URL != "" {
		t.Fatalf("expected manual image to have empty URL, got %q", image.URL)
	}
	if image.CloudInit {
		t.Fatal("expected kali-2026.1 to not support cloud-init")
	}
	if image.Category != CategorySecurity {
		t.Fatalf("expected category security, got %q", image.Category)
	}
	if image.ManualInstructions == "" {
		t.Fatal("expected manual image to have instructions")
	}
}

func TestListByCategory(t *testing.T) {
	categories := ListByCategory()

	if len(categories) == 0 {
		t.Fatal("expected at least one category")
	}

	// Verify all images appear exactly once.
	seen := make(map[string]bool)
	for cat, images := range categories {
		if cat == "" {
			t.Fatalf("category must not be empty")
		}
		for _, img := range images {
			if seen[img.Name] {
				t.Fatalf("image %q appears in multiple categories", img.Name)
			}
			seen[img.Name] = true
			if img.Category != cat {
				t.Fatalf("image %q has category %q but appears in %q", img.Name, img.Category, cat)
			}
		}
	}

	allImages := SupportedImages()
	if len(seen) != len(allImages) {
		t.Fatalf("expected %d images across categories, found %d", len(allImages), len(seen))
	}

	// Verify sorting within categories.
	for cat, images := range categories {
		sorted := make([]TrustedImage, len(images))
		copy(sorted, images)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Name < sorted[j].Name
		})
		for i := range images {
			if images[i].Name != sorted[i].Name {
				t.Fatalf("category %q is not sorted: %v", cat, images)
			}
		}
	}
}

func TestAllImagesHaveRequiredFields(t *testing.T) {
	for _, name := range SupportedImages() {
		img, ok := Lookup(name)
		if !ok {
			t.Fatalf("image %q not found in manifest", name)
		}
		if img.Name == "" {
			t.Errorf("image %q has empty Name", name)
		}
		if img.Name != name {
			t.Errorf("image key %q does not match Name %q", name, img.Name)
		}
		if img.Category == "" {
			t.Errorf("image %q has empty Category", name)
		}
		if img.Description == "" {
			t.Errorf("image %q has empty Description", name)
		}
		if img.Size == "" {
			t.Errorf("image %q has empty Size", name)
		}
		// Auto-downloadable images must have URL and SHA256.
		if img.URL != "" && img.SHA256 == "" {
			t.Errorf("image %q has URL but no SHA256", name)
		}
		if img.SHA256 != "" && img.URL == "" {
			t.Errorf("image %q has SHA256 but no URL", name)
		}
		// Manual images must have instructions.
		if img.URL == "" && img.ManualInstructions == "" {
			t.Errorf("image %q is manual (no URL) but has no ManualInstructions", name)
		}
	}
}

func TestImageNameFormat(t *testing.T) {
	for _, name := range SupportedImages() {
		if !IsSafeImageName(name) {
			t.Errorf("image name %q is not safe (doesn't match pattern)", name)
		}
	}
}

func TestGetCachedImages(t *testing.T) {
	cacheRoot := t.TempDir()

	// No cache yet.
	cached := GetCachedImages(cacheRoot)
	if len(cached) != 0 {
		t.Fatalf("expected no cached images, got %v", cached)
	}

	// Create a fake cached image.
	imgDir := filepath.Join(cacheRoot, "ubuntu-24.04")
	if err := os.MkdirAll(imgDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(imgDir, ImageFileName), []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}

	cached = GetCachedImages(cacheRoot)
	if len(cached) != 1 || cached[0] != "ubuntu-24.04" {
		t.Fatalf("expected [ubuntu-24.04], got %v", cached)
	}

	// Add another.
	imgDir2 := filepath.Join(cacheRoot, "debian-12")
	if err := os.MkdirAll(imgDir2, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(imgDir2, ImageFileName), []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}

	cached = GetCachedImages(cacheRoot)
	if len(cached) != 2 {
		t.Fatalf("expected 2 cached images, got %v", cached)
	}
	// Should be sorted.
	if cached[0] != "debian-12" || cached[1] != "ubuntu-24.04" {
		t.Fatalf("expected sorted [debian-12 ubuntu-24.04], got %v", cached)
	}
}

func TestIsCached(t *testing.T) {
	cacheRoot := t.TempDir()

	if IsCached(cacheRoot, "ubuntu-24.04") {
		t.Fatal("expected not cached before creation")
	}

	imgDir := filepath.Join(cacheRoot, "ubuntu-24.04")
	if err := os.MkdirAll(imgDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(imgDir, ImageFileName), []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}

	if !IsCached(cacheRoot, "ubuntu-24.04") {
		t.Fatal("expected cached after creation")
	}
	if IsCached(cacheRoot, "debian-12") {
		t.Fatal("expected debian-12 not cached")
	}
}
