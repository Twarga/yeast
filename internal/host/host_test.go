package host

import (
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestDetectDistroParseOSRelease(t *testing.T) {
	tests := []struct {
		content    string
		wantID     string
		wantFamily DistroFamily
	}{
		{
			content:    "ID=ubuntu\nID_LIKE=debian\nNAME=\"Ubuntu\"\nVERSION_ID=\"22.04\"\n",
			wantID:     "ubuntu",
			wantFamily: FamilyDebian,
		},
		{
			content:    "ID=debian\nNAME=\"Debian GNU/Linux\"\nVERSION_ID=\"12\"\n",
			wantID:     "debian",
			wantFamily: FamilyDebian,
		},
		{
			content:    "ID=fedora\nNAME=\"Fedora Linux\"\nVERSION_ID=\"42\"\n",
			wantID:     "fedora",
			wantFamily: FamilyFedora,
		},
		{
			content:    "ID=arch\nNAME=\"Arch Linux\"\n",
			wantID:     "arch",
			wantFamily: FamilyArch,
		},
		{
			content:    "ID=rocky\nID_LIKE=\"rhel centos fedora\"\nNAME=\"Rocky Linux\"\nVERSION_ID=\"9.3\"\n",
			wantID:     "rocky",
			wantFamily: FamilyFedora,
		},
		{
			content:    "ID=popos\nID_LIKE=ubuntu\nNAME=\"Pop!_OS\"\nVERSION_ID=\"22.04\"\n",
			wantID:     "popos",
			wantFamily: FamilyDebian,
		},
	}

	for _, tt := range tests {
		fields := parseOSRelease(tt.content)
		id := fields["ID"]
		idLike := fields["ID_LIKE"]
		family := classifyFamily(id, idLike)

		if id != tt.wantID {
			t.Errorf("content=%q: want ID=%q, got %q", tt.content, tt.wantID, id)
		}
		if family != tt.wantFamily {
			t.Errorf("content=%q: want family=%q, got %q", tt.content, tt.wantFamily, family)
		}
	}
}

func TestDistroPackageManager(t *testing.T) {
	tests := []struct {
		family  DistroFamily
		wantPM  string
	}{
		{FamilyDebian, "apt"},
		{FamilyFedora, "dnf"},
		{FamilyArch, "pacman"},
		{FamilyOpenSUSE, "zypper"},
		{FamilyUnknown, ""},
	}
	for _, tt := range tests {
		d := Distro{Family: tt.family}
		if pm := d.PackageManager(); pm != tt.wantPM {
			t.Errorf("family=%s: want PM=%q, got %q", tt.family, tt.wantPM, pm)
		}
	}
}

func TestPackageNames(t *testing.T) {
	tests := []struct {
		dep    DependencyName
		family DistroFamily
		want   []string
	}{
		{DepQEMUSystem, FamilyDebian, []string{"qemu-system-x86"}},
		{DepQEMUImg, FamilyDebian, []string{"qemu-utils"}},
		{DepISOBuilder, FamilyArch, []string{"cdrtools"}},
		{DepSSH, FamilyFedora, []string{"openssh-clients"}},
	}
	for _, tt := range tests {
		got := PackageNames(tt.dep, tt.family)
		if len(got) != len(tt.want) {
			t.Errorf("PackageNames(%s,%s): want %v, got %v", tt.dep, tt.family, tt.want, got)
			continue
		}
		for i, w := range tt.want {
			if got[i] != w {
				t.Errorf("PackageNames(%s,%s)[%d]: want %q, got %q", tt.dep, tt.family, i, w, got[i])
			}
		}
	}
}

func TestRunChecksKVMBlockerOnNativeLinux(t *testing.T) {
	env := Environment{
		Type:  EnvironmentNativeLinux,
		Arch:  ArchAMD64,
		Distro: Distro{Family: FamilyDebian},
	}

	fakeLookPath := func(name string) (string, error) {
		return "/usr/bin/" + name, nil
	}
	fakeStatPath := func(path string) (os.FileInfo, error) {
		return nil, os.ErrNotExist // KVM missing
	}

	results := RunChecks(env, fakeLookPath, fakeStatPath)

	var kvmResult *CheckResult
	for i, r := range results {
		if r.Name == DepKVM {
			kvmResult = &results[i]
			break
		}
	}
	if kvmResult == nil {
		t.Fatal("expected /dev/kvm check in results")
	}
	if kvmResult.Severity != SeverityBlocker {
		t.Errorf("native Linux with no KVM: want SeverityBlocker, got %s", kvmResult.Severity)
	}
	if kvmResult.Fix == nil {
		t.Error("expected a fix step for missing KVM on native Linux")
	}
}

func TestRunChecksKVMWarningInWSL(t *testing.T) {
	env := Environment{
		Type:  EnvironmentWSL2,
		Arch:  ArchAMD64,
		Distro: Distro{Family: FamilyDebian},
	}

	fakeLookPath := func(name string) (string, error) { return "/usr/bin/" + name, nil }
	fakeStatPath := func(path string) (os.FileInfo, error) { return nil, os.ErrNotExist }

	results := RunChecks(env, fakeLookPath, fakeStatPath)

	var kvmResult *CheckResult
	for i, r := range results {
		if r.Name == DepKVM {
			kvmResult = &results[i]
			break
		}
	}
	if kvmResult == nil {
		t.Fatal("expected /dev/kvm check")
	}
	if kvmResult.Severity != SeverityWarning {
		t.Errorf("WSL2 with no KVM: want SeverityWarning, got %s", kvmResult.Severity)
	}
}

func TestRunChecksKVMWarningWhenDeviceIsInaccessible(t *testing.T) {
	env := Environment{
		Type:   EnvironmentNativeLinux,
		Arch:   ArchAMD64,
		Distro: Distro{Family: FamilyDebian},
	}

	previousOpenKVMDevice := openKVMDevice
	defer func() {
		openKVMDevice = previousOpenKVMDevice
	}()

	openKVMDevice = func() (*os.File, error) {
		return nil, os.ErrPermission
	}

	fakeLookPath := func(name string) (string, error) { return "/usr/bin/" + name, nil }
	fakeStatPath := func(path string) (os.FileInfo, error) {
		if path == "/dev/kvm" {
			return fakeFileInfo{name: "kvm", mode: os.ModeDevice}, nil
		}
		return nil, os.ErrNotExist
	}

	results := RunChecks(env, fakeLookPath, fakeStatPath)

	var kvmResult *CheckResult
	for i, r := range results {
		if r.Name == DepKVM {
			kvmResult = &results[i]
			break
		}
	}
	if kvmResult == nil {
		t.Fatal("expected /dev/kvm check")
	}
	if kvmResult.Severity != SeverityWarning {
		t.Fatalf("present but inaccessible /dev/kvm: want SeverityWarning, got %s", kvmResult.Severity)
	}
	if kvmResult.Fix == nil {
		t.Fatal("expected a fix step for inaccessible /dev/kvm")
	}
	if !kvmResult.Fix.NeedsSudo {
		t.Fatal("expected inaccessible /dev/kvm fix to require sudo")
	}
}

func TestRunChecksISOBuilderAcceptsXorriso(t *testing.T) {
	env := Environment{
		Type:  EnvironmentNativeLinux,
		Arch:  ArchAMD64,
		Distro: Distro{Family: FamilyDebian},
	}

	fakeLookPath := func(name string) (string, error) {
		if name == "xorriso" {
			return "/usr/bin/xorriso", nil
		}
		return "/usr/bin/" + name, nil
	}
	fakeStatPath := func(path string) (os.FileInfo, error) { return nil, os.ErrNotExist }

	results := RunChecks(env, fakeLookPath, fakeStatPath)

	var isoResult *CheckResult
	for i, r := range results {
		if r.Name == DepISOBuilder {
			isoResult = &results[i]
			break
		}
	}
	if isoResult == nil {
		t.Fatal("expected iso-builder check")
	}
	if isoResult.Severity != SeverityOK {
		t.Errorf("xorriso present: want SeverityOK, got %s: %s", isoResult.Severity, isoResult.Detail)
	}
}

func TestBuildFixPlan(t *testing.T) {
	results := []CheckResult{
		{Name: DepQEMUSystem, Severity: SeverityOK},
		{
			Name:     DepSSH,
			Severity: SeverityBlocker,
			Fix: &FixStep{
				Description: "install ssh",
				Command:     []string{"apt-get", "install", "-y", "openssh-client"},
				NeedsSudo:   true,
			},
		},
		{
			Name:     DepKVM,
			Severity: SeverityWarning,
			// No fix available for container KVM
		},
	}

	plan := BuildFixPlan(results)
	if plan.Empty() {
		t.Fatal("expected non-empty fix plan")
	}
	if len(plan.Steps) != 1 {
		t.Fatalf("expected 1 fix step, got %d", len(plan.Steps))
	}
	if plan.Steps[0].Check.Name != DepSSH {
		t.Errorf("expected SSH fix step, got %s", plan.Steps[0].Check.Name)
	}
	if plan.AutomatableCount() != 1 {
		t.Errorf("expected 1 automatable step, got %d", plan.AutomatableCount())
	}
}

func TestClassifySupportTier(t *testing.T) {
	tests := []struct {
		env    EnvironmentType
		arch   Architecture
		family DistroFamily
		kvm    bool
		want   SupportTier
	}{
		{EnvironmentNativeLinux, ArchAMD64, FamilyDebian, true, SupportTierA},
		{EnvironmentNativeLinux, ArchAMD64, FamilyFedora, true, SupportTierB},
		{EnvironmentNativeLinux, ArchAMD64, FamilyArch, true, SupportTierB},
		{EnvironmentNativeLinux, ArchAMD64, FamilyDebian, false, SupportTierC},
		{EnvironmentWSL2, ArchAMD64, FamilyDebian, false, SupportTierC},
		{EnvironmentContainer, ArchAMD64, FamilyDebian, false, SupportTierC},
		{EnvironmentWSL1, ArchAMD64, FamilyDebian, false, SupportTierD},
		{EnvironmentNativeLinux, ArchARM64, FamilyDebian, true, SupportTierC},
		{EnvironmentNativeLinux, ArchUnknown, FamilyDebian, true, SupportTierC},
	}

	for _, tt := range tests {
		got := classifySupportTier(tt.env, tt.arch, Distro{Family: tt.family}, tt.kvm)
		if got != tt.want {
			t.Errorf("classifySupportTier(env=%s, arch=%s, family=%s, kvm=%v): want %s, got %s",
				tt.env, tt.arch, tt.family, tt.kvm, tt.want, got)
		}
	}
}

func TestCountSeverities(t *testing.T) {
	results := []CheckResult{
		{Severity: SeverityOK},
		{Severity: SeverityBlocker},
		{Severity: SeverityWarning},
		{Severity: SeverityBlocker},
	}
	b, w := CountSeverities(results)
	if b != 2 {
		t.Errorf("want 2 blockers, got %d", b)
	}
	if w != 1 {
		t.Errorf("want 1 warning, got %d", w)
	}
}

// Ensure the package compiles against the real exec.LookPath and os.Stat
// without invoking any real system commands.
var _ = exec.LookPath

type fakeFileInfo struct {
	name string
	mode os.FileMode
}

func (f fakeFileInfo) Name() string       { return f.name }
func (f fakeFileInfo) Size() int64        { return 0 }
func (f fakeFileInfo) Mode() os.FileMode  { return f.mode }
func (f fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (f fakeFileInfo) IsDir() bool        { return f.mode.IsDir() }
func (f fakeFileInfo) Sys() any           { return nil }
