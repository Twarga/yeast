package images

import "testing"

func TestSupportedImagesSorted(t *testing.T) {
	got := SupportedImages()
	want := []string{"ubuntu-22.04", "ubuntu-24.04"}

	if len(got) != len(want) {
		t.Fatalf("expected %d images, got %d", len(want), len(got))
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
}

func TestLookupUnknownImage(t *testing.T) {
	if _, ok := Lookup("debian-13"); ok {
		t.Fatal("expected unknown image lookup to fail")
	}
}
