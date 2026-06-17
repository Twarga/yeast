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
	DepQEMUSystem  DependencyName = "qemu-system-x86_64"
	DepQEMUImg     DependencyName = "qemu-img"
	DepISOBuilder  DependencyName = "iso-builder"
	DepSSH         DependencyName = "ssh"
	DepKVM         DependencyName = "/dev/kvm"
	DepSSHKey      DependencyName = "ssh-public-key"
	DepCacheDir    DependencyName = "cache-directory"
	DepKVMModules  DependencyName = "kvm-modules"
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

func checkKVM(env Environment, statPath func(string) (os.FileInfo, error)) CheckResult {
	info, err := statPath("/dev/kvm")
	if err != nil {
		// WSL and containers may not have KVM — treat as warning, not blocker.
		if env.Type == EnvironmentWSL2 || env.Type == EnvironmentWSL1 || env.Type == EnvironmentContainer {
			return CheckResult{
				Name:     DepKVM,
				Severity: SeverityWarning,
				Detail:   "not available; running in WSL or container — VMs will use software emulation (TCG), which is significantly slower",
			}
		}
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

	if info.Mode()&os.ModeDevice == 0 {
		return CheckResult{
			Name:     DepKVM,
			Severity: SeverityBlocker,
			Detail:   "/dev/kvm exists but is not a device node",
		}
	}

	// Device exists — check if the current user can access it.
	if err := checkKVMAccess(); err != nil {
		return CheckResult{
			Name:     DepKVM,
			Severity: SeverityWarning,
			Detail:   fmt.Sprintf("present but not accessible by current user: %v", err),
			Fix: &FixStep{
				Description: "add your user to the kvm group",
				Command:     []string{"usermod", "-aG", "kvm", currentUser()},
				NeedsSudo:   true,
				ManualOnly:  false,
			},
		}
	}

	return CheckResult{Name: DepKVM, Severity: SeverityOK, Detail: "present and accessible"}
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
