package images

import "sort"

// TrustedImage defines an allow-listed base image with a pinned URL and SHA256 hash.
type TrustedImage struct {
	Name   string
	URL    string
	SHA256 string
}

// trustedManifest is intentionally pinned to specific URLs + checksums.
// Source: https://cloud-images.ubuntu.com/releases/*/release/SHA256SUMS
// Updated: 2026-03-05
var trustedManifest = map[string]TrustedImage{
	"ubuntu-22.04": {
		Name:   "ubuntu-22.04",
		URL:    "https://cloud-images.ubuntu.com/releases/jammy/release/ubuntu-22.04-server-cloudimg-amd64.img",
		SHA256: "e66ef756881b5e682c496112201382abd76291797a7395bf81fd1bd0888f5b6f",
	},
	"ubuntu-24.04": {
		Name:   "ubuntu-24.04",
		URL:    "https://cloud-images.ubuntu.com/releases/noble/release/ubuntu-24.04-server-cloudimg-amd64.img",
		SHA256: "7aa6d9f5e8a3a55c7445b138d31a73d1187871211b2b7da9da2e1a6cbf169b21",
	},
}

func GetTrustedImage(name string) (TrustedImage, bool) {
	img, ok := trustedManifest[name]
	return img, ok
}

func SupportedImageNames() []string {
	names := make([]string, 0, len(trustedManifest))
	for name := range trustedManifest {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
