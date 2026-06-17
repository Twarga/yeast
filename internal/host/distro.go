package host

import (
	"os"
	"strings"
)

// DistroFamily groups distros by package manager / ecosystem.
type DistroFamily string

const (
	FamilyDebian  DistroFamily = "debian"  // Ubuntu, Debian, Mint, Pop
	FamilyFedora  DistroFamily = "fedora"  // Fedora, RHEL, CentOS, Rocky, Alma
	FamilyArch    DistroFamily = "arch"    // Arch, Manjaro, EndeavourOS
	FamilyOpenSUSE DistroFamily = "opensuse"
	FamilyUnknown DistroFamily = "unknown"
)

// Distro holds the parsed values from /etc/os-release.
type Distro struct {
	ID      string       // e.g. "ubuntu", "debian", "fedora"
	Name    string       // e.g. "Ubuntu", "Debian GNU/Linux"
	Version string       // e.g. "22.04", "12"
	Family  DistroFamily
}

// DetectDistro reads /etc/os-release and classifies the distro.
func DetectDistro() Distro {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return Distro{Family: FamilyUnknown}
	}
	fields := parseOSRelease(string(data))
	id := strings.ToLower(fields["ID"])
	idLike := strings.ToLower(fields["ID_LIKE"])

	d := Distro{
		ID:      id,
		Name:    fields["NAME"],
		Version: fields["VERSION_ID"],
		Family:  classifyFamily(id, idLike),
	}
	return d
}

func parseOSRelease(content string) map[string]string {
	fields := make(map[string]string)
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.IndexByte(line, '=')
		if idx < 0 {
			continue
		}
		key := line[:idx]
		val := strings.Trim(line[idx+1:], `"'`)
		fields[key] = val
	}
	return fields
}

func classifyFamily(id, idLike string) DistroFamily {
	debianIDs := []string{"ubuntu", "debian", "linuxmint", "pop", "elementary", "kali"}
	for _, v := range debianIDs {
		if id == v || strings.Contains(idLike, v) {
			return FamilyDebian
		}
	}
	fedoraIDs := []string{"fedora", "rhel", "centos", "rocky", "alma", "ol", "scientific"}
	for _, v := range fedoraIDs {
		if id == v || strings.Contains(idLike, v) {
			return FamilyFedora
		}
	}
	archIDs := []string{"arch", "manjaro", "endeavouros", "garuda"}
	for _, v := range archIDs {
		if id == v || strings.Contains(idLike, v) {
			return FamilyArch
		}
	}
	opensuseIDs := []string{"opensuse", "sles", "sled"}
	for _, v := range opensuseIDs {
		if id == v || strings.Contains(idLike, v) {
			return FamilyOpenSUSE
		}
	}
	return FamilyUnknown
}

// PackageManager returns the primary package manager for the distro family.
func (d Distro) PackageManager() string {
	switch d.Family {
	case FamilyDebian:
		return "apt"
	case FamilyFedora:
		return "dnf"
	case FamilyArch:
		return "pacman"
	case FamilyOpenSUSE:
		return "zypper"
	default:
		return ""
	}
}
