package host

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
)

// findCheck locates a single check result by name.
func findCheck(t *testing.T, results []CheckResult, name DependencyName) CheckResult {
	t.Helper()
	for _, r := range results {
		if r.Name == name {
			return r
		}
	}
	t.Fatalf("expected %s check in results", name)
	return CheckResult{}
}

func okLookPath(name string) (string, error) { return "/usr/bin/" + name, nil }

func missingKVMStat(path string) (os.FileInfo, error) { return nil, os.ErrNotExist }

// TestCheckKVMMissingIsBlockerEverywhere pins the contract: QEMU always runs
// with accel=kvm, so a missing /dev/kvm is a blocker on every host type —
// including WSL2 and containers, which previously reported a TCG warning.
func TestCheckKVMMissingIsBlockerEverywhere(t *testing.T) {
	t.Parallel()
	envs := []EnvironmentType{
		EnvironmentNativeLinux,
		EnvironmentWSL2,
		EnvironmentWSL1,
		EnvironmentContainer,
	}
	for _, envType := range envs {
		env := Environment{Type: envType, Arch: ArchAMD64, Distro: Distro{Family: FamilyDebian}}
		result := findCheck(t, RunChecks(env, okLookPath, missingKVMStat), DepKVM)
		if result.Severity != SeverityBlocker {
			t.Errorf("env=%s: missing /dev/kvm must be a blocker, got %s", envType, result.Severity)
		}
		if result.Fix == nil {
			t.Errorf("env=%s: expected a fix step for missing /dev/kvm", envType)
		}
	}
}

// TestCheckKVMNoTCGClaims guards against the old misleading wording: Yeast has
// no software-emulation fallback, so check output must never promise one.
func TestCheckKVMNoTCGClaims(t *testing.T) {
	t.Parallel()
	envs := []EnvironmentType{
		EnvironmentNativeLinux,
		EnvironmentWSL2,
		EnvironmentWSL1,
		EnvironmentContainer,
	}
	for _, envType := range envs {
		env := Environment{Type: envType, Arch: ArchAMD64, Distro: Distro{Family: FamilyDebian}}
		result := findCheck(t, RunChecks(env, okLookPath, missingKVMStat), DepKVM)
		text := result.Detail
		if result.Fix != nil {
			text += " " + result.Fix.Description + " " + result.Fix.ManualInstructions
		}
		lower := strings.ToLower(text)
		for _, banned := range []string{"tcg", "software emulation", "emulation fallback", "will fall back", "may fall back", "falls back"} {
			if strings.Contains(lower, banned) {
				t.Errorf("env=%s: kvm check must not claim %q fallback, got: %s", envType, banned, text)
			}
		}
	}
}

// TestCheckKVMWSL2Remediation verifies the WSL2 fix guidance covers the real
// failure modes: nested virtualization, the kvm group, WSL RAM, and forwarding.
func TestCheckKVMWSL2Remediation(t *testing.T) {
	t.Parallel()
	env := Environment{Type: EnvironmentWSL2, Arch: ArchAMD64, Distro: Distro{Family: FamilyDebian}}
	result := findCheck(t, RunChecks(env, okLookPath, missingKVMStat), DepKVM)

	if result.Fix == nil || !result.Fix.ManualOnly {
		t.Fatal("WSL2 missing KVM fix must be manual-only")
	}
	instructions := result.Fix.ManualInstructions
	for _, want := range []string{".wslconfig", "nestedVirtualization", "localhostForwarding", "memory", "usermod", "wsl --shutdown", "wsl --update"} {
		if !strings.Contains(instructions, want) {
			t.Errorf("WSL2 remediation missing %q:\n%s", want, instructions)
		}
	}
}

// TestCheckKVMPermissionDeniedIsBlocker simulates /dev/kvm existing but the
// user lacking read/write permission (EACCES) — a guaranteed `yeast up` failure.
func TestCheckKVMPermissionDeniedIsBlocker(t *testing.T) {
	previous := openKVMDevice
	defer func() { openKVMDevice = previous }()
	openKVMDevice = func() (*os.File, error) {
		return nil, os.ErrPermission
	}

	statKVMDevice := func(path string) (os.FileInfo, error) {
		if path == "/dev/kvm" {
			return fakeFileInfo{name: "kvm", mode: os.ModeDevice}, nil
		}
		return nil, os.ErrNotExist
	}
	env := Environment{Type: EnvironmentNativeLinux, Arch: ArchAMD64, Distro: Distro{Family: FamilyDebian}}
	result := findCheck(t, RunChecks(env, okLookPath, statKVMDevice), DepKVM)

	if result.Severity != SeverityBlocker {
		t.Fatalf("unreadable /dev/kvm must be a blocker, got %s", result.Severity)
	}
	if result.Fix == nil {
		t.Fatal("expected a fix step for permission-denied /dev/kvm")
	}
	if !result.Fix.NeedsSudo {
		t.Error("kvm group fix must require sudo")
	}
	joined := strings.Join(result.Fix.Command, " ")
	if !strings.Contains(joined, "usermod") || !strings.Contains(joined, "kvm") {
		t.Errorf("expected usermod -aG kvm fix command, got %q", joined)
	}
}

// TestCheckKVMHealthy verifies the happy path: device node present and the
// current user can open it read/write.
func TestCheckKVMHealthy(t *testing.T) {
	previous := openKVMDevice
	defer func() { openKVMDevice = previous }()

	tmp, err := os.CreateTemp(t.TempDir(), "fake-kvm")
	if err != nil {
		t.Fatalf("create fake kvm device file: %v", err)
	}
	openKVMDevice = func() (*os.File, error) { return tmp, nil }

	statKVMDevice := func(path string) (os.FileInfo, error) {
		if path == "/dev/kvm" {
			return fakeFileInfo{name: "kvm", mode: os.ModeDevice}, nil
		}
		return nil, os.ErrNotExist
	}
	env := Environment{Type: EnvironmentNativeLinux, Arch: ArchAMD64, Distro: Distro{Family: FamilyDebian}}
	result := findCheck(t, RunChecks(env, okLookPath, statKVMDevice), DepKVM)

	if result.Severity != SeverityOK {
		t.Fatalf("healthy /dev/kvm must be ok, got %s: %s", result.Severity, result.Detail)
	}
}

// TestCheckKVMNotADeviceNode covers a regular file at /dev/kvm.
func TestCheckKVMNotADeviceNode(t *testing.T) {
	t.Parallel()
	statRegularFile := func(path string) (os.FileInfo, error) {
		if path == "/dev/kvm" {
			return fakeFileInfo{name: "kvm", mode: 0}, nil
		}
		return nil, os.ErrNotExist
	}
	env := Environment{Type: EnvironmentNativeLinux, Arch: ArchAMD64, Distro: Distro{Family: FamilyDebian}}
	result := findCheck(t, RunChecks(env, okLookPath, statRegularFile), DepKVM)

	if result.Severity != SeverityBlocker {
		t.Fatalf("non-device /dev/kvm must be a blocker, got %s", result.Severity)
	}
	if !strings.Contains(result.Detail, "not a device node") {
		t.Errorf("unexpected detail: %q", result.Detail)
	}
}

// TestCheckKVMGenericOpenError verifies non-permission open errors (e.g. EBUSY)
// still surface as blockers rather than being misclassified.
func TestCheckKVMGenericOpenError(t *testing.T) {
	previous := openKVMDevice
	defer func() { openKVMDevice = previous }()
	openKVMDevice = func() (*os.File, error) {
		return nil, errors.New("device busy")
	}

	statKVMDevice := func(path string) (os.FileInfo, error) {
		if path == "/dev/kvm" {
			return fakeFileInfo{name: "kvm", mode: os.ModeDevice}, nil
		}
		return nil, os.ErrNotExist
	}
	env := Environment{Type: EnvironmentNativeLinux, Arch: ArchAMD64, Distro: Distro{Family: FamilyDebian}}
	result := findCheck(t, RunChecks(env, okLookPath, statKVMDevice), DepKVM)

	if result.Severity != SeverityBlocker {
		t.Fatalf("unopenable /dev/kvm must be a blocker, got %s", result.Severity)
	}
}

func TestOptionalCapabilityChecks(t *testing.T) {
	missing := func(string) (os.FileInfo, error) { return nil, os.ErrNotExist }
	present := func(string) (os.FileInfo, error) { return nil, nil }

	r := checkOVMF(missing)
	if r.Severity != SeverityOK || r.Name != DepOVMF {
		t.Fatalf("missing OVMF = %+v", r)
	}
	ovmfOnly := func(path string) (os.FileInfo, error) {
		if path == "/usr/share/OVMF/OVMF_CODE_4M.fd" || path == "/usr/share/OVMF/OVMF_VARS_4M.fd" {
			return nil, nil
		}
		return nil, os.ErrNotExist
	}
	if r := checkOVMF(ovmfOnly); r.Severity != SeverityOK {
		t.Fatalf("present OVMF = %+v", r)
	}

	if r := checkVirgl(missing); r.Severity != SeverityOK {
		t.Fatalf("missing render node = %+v", r)
	}
	if r := checkVirgl(present); r.Severity != SeverityOK {
		t.Fatalf("present render node = %+v", r)
	}

	notFound := func(name string) (string, error) { return "", fmt.Errorf("no %s", name) }
	if r := checkVirtiofsd(notFound, missing); r.Severity != SeverityOK {
		t.Fatalf("missing virtiofsd = %+v", r)
	}

	// capability checks are informational — a missing optional tool still
	// reports ok (the detail says "not installed (optional …)"), never warning
	// or blocker, because headless VMs don't need them
	for _, r := range []CheckResult{checkOVMF(missing), checkVirgl(missing), checkVirtiofsd(notFound, missing)} {
		if r.Severity != SeverityOK {
			t.Fatalf("optional capability %s must stay informational, got %s", r.Name, r.Severity)
		}
		if !strings.Contains(r.Detail, "optional") {
			t.Fatalf("missing-optional detail should say optional: %q", r.Detail)
		}
	}
}

func TestParseQEMUVersion(t *testing.T) {
	cases := []struct {
		out          string
		major, minor int
	}{
		{"QEMU emulator version 11.1.3 (qemu-11.1.3-1.fc45)\nCopyright", 11, 1},
		{"QEMU emulator version 8.2.0", 8, 2},
		{"QEMU emulator version 6.0.1 (Debian)", 6, 0},
		{"garbage", 0, 0},
	}
	for _, c := range cases {
		maj, min := parseQEMUVersion(c.out)
		if maj != c.major || min != c.minor {
			t.Fatalf("parseQEMUVersion(%q) = %d.%d want %d.%d", c.out, maj, min, c.major, c.minor)
		}
	}
}

func TestCheckLiveSnapshots(t *testing.T) {
	defer func() { qemuVersionOutput = func() (string, error) { return "", os.ErrNotExist } }()

	qemuVersionOutput = func() (string, error) {
		return "QEMU emulator version 11.1.3 (qemu-11.1.3-1.fc45)\n", nil
	}
	res := checkLiveSnapshots()
	if res.Severity != SeverityOK || !strings.Contains(res.Detail, "mapped-ram") {
		t.Fatalf("QEMU 11 result = %+v", res)
	}

	qemuVersionOutput = func() (string, error) {
		return "QEMU emulator version 7.0.0\n", nil
	}
	res = checkLiveSnapshots()
	if res.Severity != SeverityOK || !strings.Contains(res.Detail, "multifd") {
		t.Fatalf("QEMU 7 result = %+v", res)
	}

	qemuVersionOutput = func() (string, error) {
		return "QEMU emulator version 6.0.1\n", nil
	}
	res = checkLiveSnapshots()
	if res.Severity != SeverityOK || !strings.Contains(res.Detail, "snapshot-save") {
		t.Fatalf("QEMU 6 result = %+v", res)
	}

	qemuVersionOutput = func() (string, error) { return "", fmt.Errorf("no qemu") }
	res = checkLiveSnapshots()
	if res.Severity != SeverityOK {
		t.Fatalf("missing qemu must stay SeverityOK, got %+v", res)
	}
}
