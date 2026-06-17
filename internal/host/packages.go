package host

// PackageNames returns the distro-specific package names needed to satisfy a
// dependency. Returns nil if the dependency is not package-installable on
// this distro family.
func PackageNames(dep DependencyName, family DistroFamily) []string {
	type mapping map[DistroFamily][]string

	table := map[DependencyName]mapping{
		DepQEMUSystem: {
			FamilyDebian:  {"qemu-system-x86"},
			FamilyFedora:  {"qemu-kvm"},
			FamilyArch:    {"qemu-base"},
			FamilyOpenSUSE: {"qemu-x86"},
		},
		DepQEMUImg: {
			FamilyDebian:  {"qemu-utils"},
			FamilyFedora:  {"qemu-img"},
			FamilyArch:    {"qemu-base"},
			FamilyOpenSUSE: {"qemu-tools"},
		},
		DepISOBuilder: {
			FamilyDebian:  {"genisoimage"},
			FamilyFedora:  {"genisoimage"},
			FamilyArch:    {"cdrtools"},
			FamilyOpenSUSE: {"cdrtools"},
		},
		DepSSH: {
			FamilyDebian:  {"openssh-client"},
			FamilyFedora:  {"openssh-clients"},
			FamilyArch:    {"openssh"},
			FamilyOpenSUSE: {"openssh-clients"},
		},
		DepKVMModules: {
			// KVM modules are part of the kernel on most distros; on Debian/Ubuntu
			// the qemu-system package pulls the right meta-packages.
			FamilyDebian:  {"qemu-kvm"},
			FamilyFedora:  {},
			FamilyArch:    {},
			FamilyOpenSUSE: {},
		},
	}

	m, ok := table[dep]
	if !ok {
		return nil
	}
	pkgs, ok := m[family]
	if !ok {
		return nil
	}
	return pkgs
}

// InstallCommand returns the package install command for a given package manager.
func InstallCommand(pkgManager string, packages []string) []string {
	switch pkgManager {
	case "apt":
		cmd := []string{"apt-get", "install", "-y"}
		return append(cmd, packages...)
	case "dnf":
		cmd := []string{"dnf", "install", "-y"}
		return append(cmd, packages...)
	case "pacman":
		cmd := []string{"pacman", "-S", "--noconfirm", "--needed"}
		return append(cmd, packages...)
	case "zypper":
		cmd := []string{"zypper", "install", "-y"}
		return append(cmd, packages...)
	default:
		return nil
	}
}
