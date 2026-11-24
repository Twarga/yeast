package host

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// DependencyName identifies a host dependency Yeast requires.
type DependencyName string

const (
	DepQEMUSystem DependencyName = "qemu-system-x86_64"
	DepQEMUImg    DependencyName = "qemu-img"
	DepISOBuilder DependencyName = "iso-builder"
	DepSSH        DependencyName = "ssh"
	DepKVM        DependencyName = "/dev/kvm"
	DepSSHKey     DependencyName = "ssh-public-key"
	DepCacheDir   DependencyName = "cache-directory"
	DepKVMModules DependencyName = "kvm-modules"
	DepOVMF       DependencyName = "ovmf-uefi"
	DepVirgl      DependencyName = "virgl-3d"
	DepVirtiofsd  DependencyName = "virtiofsd"
	DepLiveSnap   DependencyName = "live-snapshots"
)

// CheckSeverity describes how bad a failing check is.
type CheckSeverity string

const (
	SeverityOK      CheckSeverity = "ok"
	SeverityWarning CheckSeverity = "warning"
	SeverityBlocker CheckSeverity = "blocker"
)

// CheckResult holds the result of a single dependency check.
type CheckResult struct {
	Name     DependencyName
	Severity CheckSeverity
	Detail   string
	Fix      *FixStep // populated if a fix is available
}

// FixStep describes a remediation action the user can take or that --fix can run.
type FixStep struct {
	Description string
	// Command is the shell command to run (may require sudo).
	Command []string
	// NeedsSudo indicates the command requires root.
	NeedsSudo bool
	// ManualOnly means the fix cannot be run automatically and needs user action.
	ManualOnly bool
	// ManualInstructions is shown when ManualOnly is true.
	ManualInstructions string
}

var openKVMDevice = func() (*os.File, error) {
	return os.OpenFile("/dev/kvm", os.O_RDWR, 0)
}

// RunChecks performs all Yeast host dependency checks for the given environment.
// The lookPath and statPath arguments allow injection for testing.
func RunChecks(
	env Environment,
	lookPath func(string) (string, error),
	statPath func(string) (os.FileInfo, error),
) []CheckResult {
	results := make([]CheckResult, 0, 8)

	results = append(results, checkBinary(DepQEMUSystem, lookPath, env.Distro))
	results = append(results, checkBinary(DepQEMUImg, lookPath, env.Distro))
	results = append(results, checkISOBuilder(lookPath, env.Distro))
	results = append(results, checkBinary(DepSSH, lookPath, env.Distro))
	results = append(results, checkKVM(env, statPath))
	results = append(results, checkOVMF(statPath))
	results = append(results, checkVirgl(statPath))
	results = append(results, checkVirtiofsd(lookPath, statPath))
	results = append(results, checkLiveSnapshots())

	return results
}

func checkBinary(dep DependencyName, lookPath func(string) (string, error), distro Distro) CheckResult {
	name := string(dep)
	path, err := lookPath(name)
	if err == nil {
		return CheckResult{Name: dep, Severity: SeverityOK, Detail: path}
	}

	pkgs := PackageNames(dep, distro.Family)
	var fix *FixStep
	if len(pkgs) > 0 && distro.PackageManager() != "" {
		cmd := InstallCommand(distro.PackageManager(), pkgs)
		fix = &FixStep{
			Description: fmt.Sprintf("install %s via %s", name, distro.PackageManager()),
			Command:     cmd,
			NeedsSudo:   true,
		}
	}

	detail := fmt.Sprintf("not found; required by Yeast")
	if len(pkgs) > 0 {
		detail = fmt.Sprintf("not found; install: %s", pkgList(pkgs))
	}

	return CheckResult{
		Name:     dep,
		Severity: SeverityBlocker,
		Detail:   detail,
		Fix:      fix,
	}
}

func checkISOBuilder(lookPath func(string) (string, error), distro Distro) CheckResult {
	for _, name := range []string{"genisoimage", "mkisofs", "xorriso"} {
		if path, err := lookPath(name); err == nil {
			return CheckResult{
				Name:     DepISOBuilder,
				Severity: SeverityOK,
				Detail:   fmt.Sprintf("%s at %s", name, path),
			}
		}
	}

	pkgs := PackageNames(DepISOBuilder, distro.Family)
	var fix *FixStep
	if len(pkgs) > 0 && distro.PackageManager() != "" {
		cmd := InstallCommand(distro.PackageManager(), pkgs)
		fix = &FixStep{
			Description: fmt.Sprintf("install ISO builder via %s", distro.PackageManager()),
			Command:     cmd,
			NeedsSudo:   true,
		}
	}

	detail := "not found; install genisoimage, mkisofs, or xorriso"
	if len(pkgs) > 0 {
		detail = fmt.Sprintf("not found; install: %s", pkgList(pkgs))
	}

	return CheckResult{
		Name:     DepISOBuilder,
		Severity: SeverityBlocker,
		Detail:   detail,
		Fix:      fix,
	}
}

// wsl2KVMRemediation is the manual fix shown when WSL2 lacks /dev/kvm.
// QEMU always runs with accel=kvm — there is no software fallback.
const wsl2KVMRemediation = `KVM is required — Yeast starts QEMU with -enable-kvm and has no slower fallback.

1. On the Windows side, edit %UserProfile%\.wslconfig and under [wsl2] add:
     nestedVirtualization=true
     localhostForwarding=true
   Also make sure the WSL "memory" setting exceeds the total RAM your yeast.yaml instances need.
2. From PowerShell run "wsl --shutdown", then reopen the distro.
3. Inside WSL verify "ls -l /dev/kvm". If it exists but is not accessible, run:
     sudo usermod -aG kvm $USER
   then restart WSL.
4. If /dev/kvm is still missing, run "wsl --update" and "wsl -l -v" to confirm the distro
   actually runs under WSL2. Nested virtualization requires Windows 11 (or a recent Windows 10 build).`

func checkKVM(env Environment, statPath func(string) (os.FileInfo, error)) CheckResult {
	info, err := statPath("/dev/kvm")
	if err != nil {
		return kvmMissingResult(env)
	}

	if info.Mode()&os.ModeDevice == 0 {
		return CheckResult{
			Name:     DepKVM,
			Severity: SeverityBlocker,
			Detail:   "/dev/kvm exists but is not a device node",
		}
	}

	// Device exists — check the current user can actually open it read/write.
	// QEMU always runs with accel=kvm, so an unreadable /dev/kvm means `yeast up` fails.
	if err := checkKVMAccess(); err != nil {
		return CheckResult{
			Name:     DepKVM,
			Severity: SeverityBlocker,
			Detail:   fmt.Sprintf("present but not readable/writable by current user: %v", err),
			Fix: &FixStep{
				Description: "add your user to the kvm group, then log out and back in",
				Command:     []string{"usermod", "-aG", "kvm", currentUser()},
				NeedsSudo:   true,
				ManualOnly:  false,
			},
		}
	}

	return CheckResult{Name: DepKVM, Severity: SeverityOK, Detail: "present and accessible"}
}

func kvmMissingResult(env Environment) CheckResult {
	switch env.Type {
	case EnvironmentWSL2:
		return CheckResult{
			Name:     DepKVM,
			Severity: SeverityBlocker,
			Detail:   "not available inside this WSL2 instance; Yeast requires KVM acceleration",
			Fix: &FixStep{
				Description:        "enable WSL2 nested virtualization",
				ManualOnly:         true,
				ManualInstructions: wsl2KVMRemediation,
			},
		}
	case EnvironmentWSL1:
		return CheckResult{
			Name:     DepKVM,
			Severity: SeverityBlocker,
			Detail:   "not available; WSL1 cannot provide /dev/kvm",
			Fix: &FixStep{
				Description:        "convert the distro to WSL2",
				ManualOnly:         true,
				ManualInstructions: "From PowerShell run `wsl --set-version <DistroName> 2`, then enable nested virtualization in .wslconfig (see the WSL2 guidance `yeast doctor` prints) and verify /dev/kvm exists.",
			},
		}
	case EnvironmentContainer:
		return CheckResult{
			Name:     DepKVM,
			Severity: SeverityBlocker,
			Detail:   "not available inside this container; Yeast requires KVM acceleration",
			Fix: &FixStep{
				Description:        "pass /dev/kvm into the container",
				ManualOnly:         true,
				ManualInstructions: "Run the container with `--device /dev/kvm` (and add your user to the kvm group), or run `yeast` directly on the host.",
			},
		}
	default:
		return CheckResult{
			Name:     DepKVM,
			Severity: SeverityBlocker,
			Detail:   "not found; enable hardware virtualization and load the KVM kernel modules before using Yeast on native Linux",
			Fix: &FixStep{
				Description:        "enable KVM on the host",
				ManualOnly:         true,
				ManualInstructions: "Enable CPU virtualization in BIOS/UEFI, then load the matching KVM modules (for example: sudo modprobe kvm_intel or sudo modprobe kvm_amd), and rerun `yeast doctor`.",
			},
		}
	}
}

func checkKVMAccess() error {
	f, err := openKVMDevice()
	if err != nil {
		return err
	}
	f.Close()
	return nil
}

func currentUser() string {
	if u := os.Getenv("SUDO_USER"); u != "" {
		return u
	}
	if u := os.Getenv("USER"); u != "" {
		return u
	}
	out, err := exec.Command("id", "-un").Output()
	if err != nil {
		return "$USER"
	}
	return strings.TrimSpace(string(out))
}

func pkgList(pkgs []string) string {
	if len(pkgs) == 1 {
		return pkgs[0]
	}
	result := ""
	for i, p := range pkgs {
		if i > 0 {
			result += " "
		}
		result += p
	}
	return result
}

// CountSeverities returns the number of blockers and warnings in a result set.
func CountSeverities(results []CheckResult) (blockers, warnings int) {
	for _, r := range results {
		switch r.Severity {
		case SeverityBlocker:
			blockers++
		case SeverityWarning:
			warnings++
		}
	}
	return blockers, warnings
}

// --- optional capability checks (informational, never blockers) -------------

// ovmfCandidateFiles mirrors the QEMU runtime's search order; keep in sync
// with internal/runtime/qemu/firmware.go.
var ovmfCandidateFiles = [][2]string{
	{"/usr/share/OVMF/OVMF_CODE_4M.fd", "/usr/share/OVMF/OVMF_VARS_4M.fd"},
	{"/usr/share/OVMF/OVMF_CODE.fd", "/usr/share/OVMF/OVMF_VARS.fd"},
	{"/usr/share/edk2/ovmf/OVMF_CODE.fd", "/usr/share/edk2/ovmf/OVMF_VARS.fd"},
	{"/usr/share/edk2/ovmf/OVMF_CODE_4M.secboot.fd", "/usr/share/edk2/ovmf/OVMF_VARS_4M.fd"},
	{"/usr/share/edk2-ovmf/x64/OVMF_CODE.fd", "/usr/share/edk2-ovmf/x64/OVMF_VARS.fd"},
	{"/usr/share/qemu/OVMF_CODE.fd", "/usr/share/qemu/OVMF_VARS.fd"},
}

func checkOVMF(statPath func(string) (os.FileInfo, error)) CheckResult {
	for _, pair := range ovmfCandidateFiles {
		if _, err := statPath(pair[0]); err != nil {
			continue
		}
		if _, err := statPath(pair[1]); err != nil {
			continue
		}
		return CheckResult{Name: DepOVMF, Severity: SeverityOK, Detail: pair[0]}
	}
	return CheckResult{
		Name:     DepOVMF,
		Severity: SeverityOK,
		Detail:   "not installed (optional — only needed for firmware: uefi); install ovmf/edk2-ovmf",
	}
}

func checkVirgl(statPath func(string) (os.FileInfo, error)) CheckResult {
	if _, err := statPath("/dev/dri/renderD128"); err == nil {
		return CheckResult{Name: DepVirgl, Severity: SeverityOK, Detail: "/dev/dri/renderD128 present (Wayland 3D acceleration available)"}
	}
	return CheckResult{
		Name:     DepVirgl,
		Severity: SeverityOK,
		Detail:   "no /dev/dri/renderD128 (optional — only needed for display.gpu: virgl)",
	}
}

func checkVirtiofsd(lookPath func(string) (string, error), statPath func(string) (os.FileInfo, error)) CheckResult {
	for _, candidate := range []string{"/usr/libexec/virtiofsd", "/usr/lib/qemu/virtiofsd", "/usr/libexec/virtiofsd-rs"} {
		if _, err := statPath(candidate); err == nil {
			return CheckResult{Name: DepVirtiofsd, Severity: SeverityOK, Detail: candidate}
		}
	}
	if path, err := lookPath("virtiofsd"); err == nil {
		return CheckResult{Name: DepVirtiofsd, Severity: SeverityOK, Detail: path}
	}
	return CheckResult{
		Name:     DepVirtiofsd,
		Severity: SeverityOK,
		Detail:   "not installed (optional — shared mounts fall back to 9p automatically)",
	}
}

// qemuVersionOutput runs `qemu-system-x86_64 --version`; injectable for tests.
var qemuVersionOutput = func() (string, error) {
	out, err := exec.Command("qemu-system-x86_64", "--version").Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// checkLiveSnapshots reports which live-snapshot method the host QEMU
// supports. Always SeverityOK — informational only, never a blocker:
// stopped-VM overlay snapshots work on any QEMU, and live snapshots fall
// back to slower methods on old versions.
//
// Version thresholds (upstream QEMU release notes):
//   - QEMU ≥ 8.2  → mapped-ram (fast file: RAM migration) + multifd
//   - QEMU ≥ 7.0  → multifd only
//   - QEMU ≥ 6.0  → snapshot-save/snapshot-load internal-snapshot jobs
//   - older       → stopped-VM overlay snapshots only
func checkLiveSnapshots() CheckResult {
	out, err := qemuVersionOutput()
	if err != nil {
		return CheckResult{
			Name:     DepLiveSnap,
			Severity: SeverityOK,
			Detail:   "qemu version unknown (optional — stopped snapshots still work)",
		}
	}
	major, minor := parseQEMUVersion(out)
	switch {
	case major > 8 || (major == 8 && minor >= 2):
		return CheckResult{
			Name:     DepLiveSnap,
			Severity: SeverityOK,
			Detail:   fmt.Sprintf("QEMU %d.%d — live snapshots via mapped-ram file migration (fast)", major, minor),
		}
	case major > 7 || (major == 7 && minor >= 0):
		return CheckResult{
			Name:     DepLiveSnap,
			Severity: SeverityOK,
			Detail:   fmt.Sprintf("QEMU %d.%d — live snapshots via multifd file migration", major, minor),
		}
	case major >= 6:
		return CheckResult{
			Name:     DepLiveSnap,
			Severity: SeverityOK,
			Detail:   fmt.Sprintf("QEMU %d.%d — live snapshots via snapshot-save job (slower)", major, minor),
		}
	default:
		return CheckResult{
			Name:     DepLiveSnap,
			Severity: SeverityOK,
			Detail:   fmt.Sprintf("QEMU %d.%d — only stopped-VM overlay snapshots supported", major, minor),
		}
	}
}

// parseQEMUVersion extracts major.minor from `qemu-system-x86_64 --version`
// output ("QEMU emulator version 11.1.3 (...)").
func parseQEMUVersion(out string) (major, minor int) {
	idx := strings.Index(out, "version ")
	if idx < 0 {
		return 0, 0
	}
	var majorF, minorF int
	if _, err := fmt.Sscanf(out[idx:], "version %d.%d", &majorF, &minorF); err != nil {
		return 0, 0
	}
	return majorF, minorF
}
