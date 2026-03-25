package images

import "sort"

// TrustedImage defines an allow-listed base image with a pinned URL and SHA256 hash.
type TrustedImage struct {
	Name   string
	URL    string
	SHA256 string
}

// trustedManifest is intentionally pinned to immutable release URLs + checksums.
// Source:
// - https://cloud-images.ubuntu.com/releases/jammy/release-20260320/SHA256SUMS
// - https://cloud-images.ubuntu.com/releases/noble/release-20260321/SHA256SUMS
// Updated: 2026-03-25
var trustedManifest = map[string]TrustedImage{
	"ubuntu-22.04": {
		Name:   "ubuntu-22.04",
		URL:    "https://cloud-images.ubuntu.com/releases/jammy/release-20260320/ubuntu-22.04-server-cloudimg-amd64.img",
		SHA256: "ea85b16f81b3f6aa53a1260912d3f991fc33e0e0fc1d73f0b8c9c96247e42fdb",
	},
	"ubuntu-24.04": {
		Name:   "ubuntu-24.04",
		URL:    "https://cloud-images.ubuntu.com/releases/noble/release-20260321/ubuntu-24.04-server-cloudimg-amd64.img",
		SHA256: "5c3ddb00f60bc455dac0862fabe9d8bacec46c33ac1751143c5c3683404b110d",
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
