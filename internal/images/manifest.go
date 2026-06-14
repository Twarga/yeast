package images

import (
	"os"
	"path/filepath"
	"sort"
)

type ImageCategory string

const (
	CategoryGeneral    ImageCategory = "general"
	CategoryEnterprise ImageCategory = "enterprise"
	CategoryDevOps     ImageCategory = "devops"
	CategorySecurity   ImageCategory = "security"
	CategoryMinimal    ImageCategory = "minimal"
	CategoryNiche      ImageCategory = "niche"
)

type TrustedImage struct {
	Name        string
	URL         string
	Checksum    string
	Category    ImageCategory
	Description string
	CloudInit   bool
	Size        string
	// ManualInstructions is set for images that require manual download/setup.
	// When non-empty, the image is searchable but not auto-downloadable.
	ManualInstructions string
}

var trustedManifest = map[string]TrustedImage{
	// ── General Purpose ──────────────────────────────────────────────
	"ubuntu-24.04": {
		Name:        "ubuntu-24.04",
		URL:         "https://cloud-images.ubuntu.com/releases/noble/release-20260321/ubuntu-24.04-server-cloudimg-amd64.img",
		Checksum:    "5c3ddb00f60bc455dac0862fabe9d8bacec46c33ac1751143c5c3683404b110d",
		Category:    CategoryGeneral,
		Description: "Ubuntu 24.04 LTS — default choice for web dev, containers, DevOps",
		CloudInit:   true,
		Size:        "~600MB",
	},
	"ubuntu-22.04": {
		Name:        "ubuntu-22.04",
		URL:         "https://cloud-images.ubuntu.com/releases/jammy/release-20260320/ubuntu-22.04-server-cloudimg-amd64.img",
		Checksum:    "ea85b16f81b3f6aa53a1260912d3f991fc33e0e0fc1d73f0b8c9c96247e42fdb",
		Category:    CategoryGeneral,
		Description: "Ubuntu 22.04 LTS — legacy LTS, stability-critical workloads",
		CloudInit:   true,
		Size:        "~500MB",
	},
	"debian-12": {
		Name:        "debian-12",
		URL:         "https://cloud.debian.org/images/cloud/bookworm/latest/debian-12-generic-amd64.qcow2",
		Checksum:    "c39495a4d81927810e93f825b0b135a08e798f46dc748e78d6bfa2984f93ffc4bcfdc7a6b7f97666ad66fbae3e0f21debd24466ce1d844d0de7856fdc7537d58",
		Category:    CategoryGeneral,
		Description: "Debian 12 (Bookworm) — rock-solid stability, minimal attack surface",
		CloudInit:   true,
		Size:        "~400MB",
	},
	"debian-13": {
		Name:        "debian-13",
		URL:         "https://cloud.debian.org/images/cloud/trixie/latest/debian-13-generic-amd64.qcow2",
		Checksum:    "97675b27e69153002c4e13644e36200c8f9067f661dca00918c54f1cacbdb88d4bff8c0fbf5cf5d63a0397bdf0cc472d7a6372bae5281bf7ced756249c10f8a2",
		Category:    CategoryGeneral,
		Description: "Debian 13 (Trixie) — testing release, newer packages",
		CloudInit:   true,
		Size:        "~400MB",
	},

	// ── DevOps & Cloud ───────────────────────────────────────────────
	"fedora-42": {
		Name:        "fedora-42",
		URL:         "https://download.fedoraproject.org/pub/fedora/linux/releases/42/Cloud/x86_64/images/Fedora-Cloud-Base-Generic-42-1.1.x86_64.qcow2",
		Checksum:    "e401a4db2e5e04d1967b6729774faa96da629bcf3ba90b67d8d9cce9906bec0f",
		Category:    CategoryDevOps,
		Description: "Fedora 42 — cutting-edge tooling, first to ship new kernels",
		CloudInit:   true,
		Size:        "~500MB",
	},
	"fedora-41": {
		Name:        "fedora-41",
		URL:         "https://download.fedoraproject.org/pub/fedora/linux/releases/41/Cloud/x86_64/images/Fedora-Cloud-Base-Generic-41-1.4.x86_64.qcow2",
		Checksum:    "6205ae0c524b4d1816dbd3573ce29b5c44ed26c9fbc874fbe48c41c89dd0bac2",
		Category:    CategoryDevOps,
		Description: "Fedora 41 — balanced stable/bleeding-edge",
		CloudInit:   true,
		Size:        "~500MB",
	},

	// ── Enterprise ───────────────────────────────────────────────────
	"rocky-9": {
		Name:        "rocky-9",
		URL:         "https://dl.rockylinux.org/pub/rocky/9/images/x86_64/Rocky-9-GenericCloud-Base-latest.x86_64.qcow2",
		Checksum:    "92c206cc6f790c61583247eefe87890f8828420662c17cacf247cec78ab4eec8",
		Category:    CategoryEnterprise,
		Description: "Rocky Linux 9 — RHEL-compatible, enterprise production",
		CloudInit:   true,
		Size:        "~1GB",
	},
	"alma-9": {
		Name:        "alma-9",
		URL:         "https://repo.almalinux.org/almalinux/9/cloud/x86_64/images/AlmaLinux-9-GenericCloud-latest.x86_64.qcow2",
		Checksum:    "c397eed7023e92c841155831b1f47e26300e5bef0f0256c129322307c897a251",
		Category:    CategoryEnterprise,
		Description: "AlmaLinux 9 — RHEL-compatible, community-driven",
		CloudInit:   true,
		Size:        "~1GB",
	},
	"centos-stream-9": {
		Name:        "centos-stream-9",
		URL:         "https://cloud.centos.org/centos/9-stream/x86_64/images/CentOS-Stream-GenericCloud-9-latest.x86_64.qcow2",
		Checksum:    "459ed0b4a3762130c5f3c4a9c8124e9f0ab3fa176bbf5384d21ed3413c088649",
		Category:    CategoryEnterprise,
		Description: "CentOS Stream 9 — upstream RHEL, CI/CD testing",
		CloudInit:   true,
		Size:        "~800MB",
	},
	"amazon-linux-2023": {
		Name:        "amazon-linux-2023",
		URL:         "",
		Checksum:    "",
		Category:    CategoryEnterprise,
		Description: "Amazon Linux 2023 — AWS-optimized, RHEL-compatible",
		CloudInit:   true,
		Size:        "~1.4GB",
		ManualInstructions: `Amazon Linux 2023 requires manual download:

  1. Download QCOW2 image from:
     https://cdn.amazonlinux.com/al2023/os-images/latest/kvm/

  2. Place the qcow2 file:
     mkdir -p ~/.yeast/cache/images/amazon-linux-2023/
     mv al2023-kvm-*.qcow2 ~/.yeast/cache/images/amazon-linux-2023/image.qcow2

  3. Re-run: yeast up

  Note: cloud-init is supported — provisioning will work after manual setup`,
	},

	// ── Security ─────────────────────────────────────────────────────
	"kali-2026.1": {
		Name:        "kali-2026.1",
		URL:         "",
		Checksum:    "",
		Category:    CategorySecurity,
		Description: "Kali Linux 2026.1 — 600+ pentesting tools, default creds: kali/kali",
		CloudInit:   false,
		Size:        "~3.6GB",
		ManualInstructions: `Kali Linux requires manual download:

  1. Download QEMU image from:
     https://cdimage.kali.org/kali-2026.1/kali-linux-2026.1-qemu-amd64.7z

  2. Extract the archive:
     7z x kali-linux-2026.1-qemu-amd64.7z

  3. Place the qcow2 file:
     mkdir -p ~/.yeast/cache/images/kali-2026.1/
     mv kali-linux-2026.1-qemu-amd64.qcow2 ~/.yeast/cache/images/kali-2026.1/image.qcow2

  4. Re-run: yeast up

  Note: Default credentials are kali/kali
  Note: cloud-init is not supported — provisioning will be limited`,
	},
	"parrot-security-7.1": {
		Name:        "parrot-security-7.1",
		URL:         "",
		Checksum:    "",
		Category:    CategorySecurity,
		Description: "Parrot Security 7.1 — 800+ security tools, forensics, reverse engineering",
		CloudInit:   false,
		Size:        "~11.7GB",
		ManualInstructions: `Parrot Security requires manual download:

  1. Download QCOW2 image from:
     https://parrotsec.org/download/?edition=security
     or mirrors: https://mirrors.mit.edu/parrot/iso/7.1/

  2. Place the qcow2 file:
     mkdir -p ~/.yeast/cache/images/parrot-security-7.1/
     mv Parrot-security-7.1_amd64.qcow2 ~/.yeast/cache/images/parrot-security-7.1/image.qcow2

  3. Re-run: yeast up

  Note: cloud-init is not supported — provisioning will be limited`,
	},

	// ── Minimal ──────────────────────────────────────────────────────
	"alpine-3.21": {
		Name:        "alpine-3.21",
		URL:         "",
		Checksum:    "",
		Category:    CategoryMinimal,
		Description: "Alpine Linux 3.21 — minimal (~50MB), fastest boot, containers",
		CloudInit:   false,
		Size:        "~50MB",
		ManualInstructions: `Alpine Linux requires manual setup:

  1. Download virt ISO from:
     https://dl-cdn.alpinelinux.org/alpine/v3.21/releases/x86_64/

  2. Convert ISO to QCOW2:
     qemu-img convert -f iso -O qcow2 alpine-virt-3.21.0-x86_64.iso ~/.yeast/cache/images/alpine-3.21/image.qcow2

  3. Or use the standard installer ISO for a full install

  4. Re-run: yeast up

  Note: cloud-init is not supported — manual configuration required`,
	},

	// ── Niche ────────────────────────────────────────────────────────
	"arch-linux": {
		Name:        "arch-linux",
		URL:         "",
		Checksum:    "",
		Category:    CategoryNiche,
		Description: "Arch Linux — rolling release, bleeding-edge, DIY",
		CloudInit:   false,
		Size:        "~800MB",
		ManualInstructions: `Arch Linux requires manual setup:

  1. Download QCOW2 from arch-boxes project:
     https://gitlab.archlinux.org/archlinux/arch-boxes/-/packages

  2. Or build your own with archiso

  3. Place the qcow2 file:
     mkdir -p ~/.yeast/cache/images/arch-linux/
     mv arch-linux.qcow2 ~/.yeast/cache/images/arch-linux/image.qcow2

  4. Re-run: yeast up

  Note: Rolling release — no version pinning
  Note: cloud-init is not supported (arch-boxes images include it)`,
	},
	"nixos-24.11": {
		Name:        "nixos-24.11",
		URL:         "",
		Checksum:    "",
		Category:    CategoryNiche,
		Description: "NixOS 24.11 — declarative config, reproducible builds",
		CloudInit:   false,
		Size:        "~1GB",
		ManualInstructions: `NixOS requires manual setup:

  1. Build a QCOW2 image with nixos-generators:
     nix run github:nix-community/nixos-generators -- --format qcow2

  2. Or download from: https://nixos.org/download/

  3. Place the qcow2 file:
     mkdir -p ~/.yeast/cache/images/nixos-24.11/
     mv nixos.qcow2 ~/.yeast/cache/images/nixos-24.11/image.qcow2

  4. Re-run: yeast up

  Note: Declarative configuration — edit /etc/nixos/configuration.nix
  Note: cloud-init is not supported`,
	},
	"opensuse-leap-15.6": {
		Name:        "opensuse-leap-15.6",
		URL:         "",
		Checksum:    "",
		Category:    CategoryEnterprise,
		Description: "openSUSE Leap 15.6 — SLES-compatible, enterprise, SAP environments",
		CloudInit:   false,
		Size:        "~1GB",
		ManualInstructions: `openSUSE Leap requires manual setup:

  1. Download QCOW2 or convert from ISO:
     https://download.opensuse.org/distribution/leap/15.6/iso/

  2. Convert ISO to QCOW2:
     qemu-img convert -f iso -O qcow2 openSUSE-Leap-15.6-DVD-x86_64-Media.iso ~/.yeast/cache/images/opensuse-leap-15.6/image.qcow2

  3. Re-run: yeast up

  Note: cloud-init is not supported — manual configuration required`,
	},
}

func SupportedImages() []string {
	names := make([]string, 0, len(trustedManifest))
	for name := range trustedManifest {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func Lookup(name string) (TrustedImage, bool) {
	image, ok := trustedManifest[name]
	return image, ok
}

// ListByCategory returns all images grouped by category, sorted alphabetically within each group.
func ListByCategory() map[ImageCategory][]TrustedImage {
	result := make(map[ImageCategory][]TrustedImage)
	for _, img := range trustedManifest {
		result[img.Category] = append(result[img.Category], img)
	}
	for cat := range result {
		sort.Slice(result[cat], func(i, j int) bool {
			return result[cat][i].Name < result[cat][j].Name
		})
	}
	return result
}

// IsCached checks whether an image is present in the local cache.
func IsCached(cacheRoot, imageName string) bool {
	paths, err := ResolveCachePaths(cacheRoot, imageName)
	if err != nil {
		return false
	}
	_, err = os.Stat(paths.ImageFile)
	return err == nil
}

// GetCachedImages scans the cache root and returns names of all cached images.
func GetCachedImages(cacheRoot string) []string {
	entries, err := os.ReadDir(cacheRoot)
	if err != nil {
		return nil
	}
	var cached []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		imgFile := filepath.Join(cacheRoot, entry.Name(), ImageFileName)
		if _, err := os.Stat(imgFile); err == nil {
			cached = append(cached, entry.Name())
		}
	}
	sort.Strings(cached)
	return cached
}
