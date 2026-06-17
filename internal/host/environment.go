// Package host provides host environment detection, dependency checks,
// remediation planning, and fix execution for Yeast.
package host

import (
	"os"
	"strings"
)

// EnvironmentType identifies the execution context.
type EnvironmentType string

const (
	EnvironmentNativeLinux EnvironmentType = "native-linux"
	EnvironmentWSL2        EnvironmentType = "wsl2"
	EnvironmentWSL1        EnvironmentType = "wsl1"
	EnvironmentContainer   EnvironmentType = "container"
)

// SupportTier communicates the level of product support for the current host.
type SupportTier string

const (
	SupportTierA SupportTier = "A" // x86_64 Ubuntu/Debian native with KVM
	SupportTierB SupportTier = "B" // x86_64 Fedora/Arch native with KVM
	SupportTierC SupportTier = "C" // WSL2, containers, other distros
	SupportTierD SupportTier = "D" // unsupported (WSL1, unknown arch)
)

// Architecture identifies the CPU architecture.
type Architecture string

const (
	ArchAMD64   Architecture = "amd64"
	ArchARM64   Architecture = "arm64"
	ArchUnknown Architecture = "unknown"
)

// Environment describes the host execution environment.
type Environment struct {
	Type         EnvironmentType
	Arch         Architecture
	SupportTier  SupportTier
	Distro       Distro
	KVMAvailable bool
}

// Detect inspects the current host and returns an Environment.
// It reads /proc/version, /etc/os-release, and related pseudo-files.
func Detect() Environment {
	arch := detectArch()
	envType := detectEnvironmentType()
	distro := DetectDistro()
	kvmOK := kvmDevicePresent()

	tier := classifySupportTier(envType, arch, distro, kvmOK)

	return Environment{
		Type:         envType,
		Arch:         arch,
		SupportTier:  tier,
		Distro:       distro,
		KVMAvailable: kvmOK,
	}
}

func detectArch() Architecture {
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return ArchUnknown
	}
	content := string(data)
	if strings.Contains(content, "x86_64") || strings.Contains(content, "GenuineIntel") || strings.Contains(content, "AuthenticAMD") {
		return ArchAMD64
	}
	// uname -m is more reliable; approximate via /proc/cpuinfo model name
	if strings.Contains(content, "aarch64") || strings.Contains(content, "ARM") {
		return ArchARM64
	}
	// Fall back to uname-derived value from proc
	data2, err := os.ReadFile("/proc/sys/kernel/arch")
	if err == nil {
		arch := strings.TrimSpace(string(data2))
		switch arch {
		case "x86_64":
			return ArchAMD64
		case "aarch64":
			return ArchARM64
		}
	}
	return ArchUnknown
}

func detectEnvironmentType() EnvironmentType {
	// Container detection first.
	if isContainer() {
		return EnvironmentContainer
	}
	// WSL detection.
	data, err := os.ReadFile("/proc/version")
	if err == nil {
		lower := strings.ToLower(string(data))
		if strings.Contains(lower, "microsoft") || strings.Contains(lower, "wsl") {
			// Distinguish WSL1 vs WSL2 via kernel osrelease.
			release, releaseErr := os.ReadFile("/proc/sys/kernel/osrelease")
			if releaseErr == nil && strings.Contains(string(release), "WSL2") {
				return EnvironmentWSL2
			}
			return EnvironmentWSL1
		}
	}
	return EnvironmentNativeLinux
}

func isContainer() bool {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	data, err := os.ReadFile("/proc/self/cgroup")
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "docker") || strings.Contains(string(data), "lxc")
}

func kvmDevicePresent() bool {
	info, err := os.Stat("/dev/kvm")
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeDevice != 0
}

func classifySupportTier(env EnvironmentType, arch Architecture, distro Distro, kvmOK bool) SupportTier {
	if env == EnvironmentWSL1 {
		return SupportTierD
	}
	if env == EnvironmentWSL2 || env == EnvironmentContainer {
		return SupportTierC
	}
	if arch != ArchAMD64 {
		return SupportTierC
	}
	if !kvmOK {
		return SupportTierC
	}
	switch distro.Family {
	case FamilyDebian:
		return SupportTierA
	case FamilyFedora, FamilyArch:
		return SupportTierB
	default:
		return SupportTierC
	}
}

// SupportTierLabel returns a human-readable description of the support tier.
func (t SupportTier) Label() string {
	switch t {
	case SupportTierA:
		return "first-class (Ubuntu/Debian native x86_64 with KVM)"
	case SupportTierB:
		return "supported (Fedora/Arch native x86_64 with KVM)"
	case SupportTierC:
		return "beta / best-effort (WSL, container, non-primary distro, or missing KVM)"
	case SupportTierD:
		return "unsupported (WSL1 or unknown architecture)"
	default:
		return string(t)
	}
}
