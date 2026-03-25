package images

import (
	"regexp"
	"testing"
)

func TestTrustedManifestUsesImmutableReleaseURLs(t *testing.T) {
	immutableReleaseURL := regexp.MustCompile(`^https://cloud-images\.ubuntu\.com/releases/[a-z0-9-]+/release-[0-9]{8}(?:\.[0-9]+)?/.+\.img$`)
	sha256Hex := regexp.MustCompile(`^[0-9a-f]{64}$`)

	for name, spec := range trustedManifest {
		if spec.Name != name {
			t.Fatalf("manifest key %q does not match image name %q", name, spec.Name)
		}
		if !immutableReleaseURL.MatchString(spec.URL) {
			t.Fatalf("trusted image %q does not use an immutable release URL: %s", name, spec.URL)
		}
		if !sha256Hex.MatchString(spec.SHA256) {
			t.Fatalf("trusted image %q has invalid SHA256: %s", name, spec.SHA256)
		}
	}
}
