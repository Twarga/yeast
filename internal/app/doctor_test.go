package app

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"yeast/internal/host"
	"yeast/internal/provision/cloudinit"
)

func TestDoctorReportsHealthyHost(t *testing.T) {
	previousLookPath := lookPath
	previousStatPath := statPath
	previousDetectHostEnvironment := detectHostEnvironment
	previousRunHostChecks := runHostChecks
	defer func() {
		lookPath = previousLookPath
		statPath = previousStatPath
		detectHostEnvironment = previousDetectHostEnvironment
		runHostChecks = previousRunHostChecks
	}()

	lookPath = func(file string) (string, error) {
		return "/usr/bin/" + file, nil
	}
	statPath = func(path string) (os.FileInfo, error) {
		if path == "/dev/kvm" {
			return fakeFileInfo{name: "kvm", mode: os.ModeDevice}, nil
		}
		if strings.HasSuffix(path, filepath.Join(".yeast", "cache", "images")) {
			return fakeFileInfo{name: "images", mode: os.ModeDir}, nil
		}
		return nil, os.ErrNotExist
	}
	detectHostEnvironment = func() host.Environment {
		return host.Environment{
			Type:        host.EnvironmentNativeLinux,
			Arch:        host.ArchAMD64,
			SupportTier: host.SupportTierA,
			Distro:      host.Distro{Family: host.FamilyDebian},
		}
	}
	runHostChecks = func(env host.Environment, lookPath func(string) (string, error), statPath func(string) (os.FileInfo, error)) []host.CheckResult {
		return []host.CheckResult{
			{Name: host.DepQEMUSystem, Severity: host.SeverityOK, Detail: "/usr/bin/qemu-system-x86_64"},
			{Name: host.DepQEMUImg, Severity: host.SeverityOK, Detail: "/usr/bin/qemu-img"},
			{Name: host.DepISOBuilder, Severity: host.SeverityOK, Detail: "xorriso at /usr/bin/xorriso"},
			{Name: host.DepSSH, Severity: host.SeverityOK, Detail: "/usr/bin/ssh"},
			{Name: host.DepKVM, Severity: host.SeverityOK, Detail: "present and accessible"},
		}
	}

	service := NewService()
	service.discoverSSHKey = func() (string, error) { return "ssh-ed25519 AAAA", nil }
	service.resolveYeastHome = func() (string, error) { return "/home/test/.yeast", nil }

	result, err := service.Doctor()
	if err != nil {
		t.Fatalf("Doctor returned error: %v", err)
	}
	if result.Blockers != 0 {
		t.Fatalf("expected no blockers, got %d", result.Blockers)
	}
	if result.Warnings != 0 {
		t.Fatalf("expected no warnings, got %d", result.Warnings)
	}
	if len(result.Checks) != 7 {
		t.Fatalf("expected 7 checks, got %d", len(result.Checks))
	}
}

func TestDoctorReportsExpectedBlockersAndWarnings(t *testing.T) {
	previousLookPath := lookPath
	previousStatPath := statPath
	previousDetectHostEnvironment := detectHostEnvironment
	previousRunHostChecks := runHostChecks
	defer func() {
		lookPath = previousLookPath
		statPath = previousStatPath
		detectHostEnvironment = previousDetectHostEnvironment
		runHostChecks = previousRunHostChecks
	}()

	lookPath = func(file string) (string, error) {
		if file == "ssh" {
			return "/usr/bin/ssh", nil
		}
		return "", errors.New("missing")
	}
	statPath = func(path string) (os.FileInfo, error) {
		return nil, os.ErrNotExist
	}
	detectHostEnvironment = func() host.Environment {
		return host.Environment{
			Type:        host.EnvironmentNativeLinux,
			SupportTier: host.SupportTierC,
			Distro:      host.Distro{Family: host.FamilyDebian},
		}
	}
	runHostChecks = host.RunChecks

	service := NewService()
	service.discoverSSHKey = func() (string, error) { return "", cloudinit.ErrNoSSHPublicKey }
	service.resolveYeastHome = func() (string, error) { return "/home/test/.yeast", nil }

	result, err := service.Doctor()
	if err != nil {
		t.Fatalf("Doctor returned error: %v", err)
	}
	if result.Blockers != 5 {
		t.Fatalf("expected 5 blockers, got %d", result.Blockers)
	}
	if result.Warnings != 1 {
		t.Fatalf("expected 1 warning, got %d", result.Warnings)
	}
}

func TestDoctorKVMWarningInWSL(t *testing.T) {
	previousLookPath := lookPath
	previousStatPath := statPath
	previousDetectHostEnvironment := detectHostEnvironment
	previousRunHostChecks := runHostChecks
	defer func() {
		lookPath = previousLookPath
		statPath = previousStatPath
		detectHostEnvironment = previousDetectHostEnvironment
		runHostChecks = previousRunHostChecks
	}()

	lookPath = func(file string) (string, error) { return "/usr/bin/" + file, nil }
	statPath = func(path string) (os.FileInfo, error) { return nil, os.ErrNotExist }
	detectHostEnvironment = func() host.Environment {
		return host.Environment{
			Type:        host.EnvironmentWSL2,
			SupportTier: host.SupportTierC,
			Distro:      host.Distro{Family: host.FamilyDebian},
		}
	}
	runHostChecks = func(env host.Environment, lookPath func(string) (string, error), statPath func(string) (os.FileInfo, error)) []host.CheckResult {
		return []host.CheckResult{
			{
				Name:     host.DepQEMUSystem,
				Severity: host.SeverityOK,
				Detail:   "/usr/bin/qemu-system-x86_64",
			},
			{
				Name:     host.DepQEMUImg,
				Severity: host.SeverityOK,
				Detail:   "/usr/bin/qemu-img",
			},
			{
				Name:     host.DepISOBuilder,
				Severity: host.SeverityOK,
				Detail:   "xorriso at /usr/bin/xorriso",
			},
			{
				Name:     host.DepSSH,
				Severity: host.SeverityOK,
				Detail:   "/usr/bin/ssh",
			},
			{
				Name:     host.DepKVM,
				Severity: host.SeverityWarning,
				Detail:   "not available; running in WSL or container — VMs will use software emulation (TCG), which is significantly slower",
			},
		}
	}

	service := NewService()
	service.discoverSSHKey = func() (string, error) { return "ssh-ed25519 AAAA", nil }
	service.resolveYeastHome = func() (string, error) { return "/home/test/.yeast", nil }

	result, err := service.Doctor()
	if err != nil {
		t.Fatalf("Doctor returned error: %v", err)
	}

	var kvmCheck *DoctorCheck
	for i, c := range result.Checks {
		if c.Name == "/dev/kvm" {
			kvmCheck = &result.Checks[i]
			break
		}
	}
	if kvmCheck == nil {
		t.Fatal("expected /dev/kvm check")
	}
	if kvmCheck.Status != CheckStatusWarning {
		t.Fatalf("expected KVM to be a warning in WSL, got %s", kvmCheck.Status)
	}
}

func TestDoctorUsesHostChecksForInaccessibleKVM(t *testing.T) {
	previousDetectHostEnvironment := detectHostEnvironment
	previousRunHostChecks := runHostChecks
	defer func() {
		detectHostEnvironment = previousDetectHostEnvironment
		runHostChecks = previousRunHostChecks
	}()

	detectHostEnvironment = func() host.Environment {
		return host.Environment{
			Type:        host.EnvironmentContainer,
			SupportTier: host.SupportTierC,
			Distro:      host.Distro{Family: host.FamilyDebian},
		}
	}
	runHostChecks = func(env host.Environment, lookPath func(string) (string, error), statPath func(string) (os.FileInfo, error)) []host.CheckResult {
		return []host.CheckResult{
			{
				Name:     host.DepQEMUSystem,
				Severity: host.SeverityOK,
				Detail:   "/usr/bin/qemu-system-x86_64",
			},
			{
				Name:     host.DepQEMUImg,
				Severity: host.SeverityOK,
				Detail:   "/usr/bin/qemu-img",
			},
			{
				Name:     host.DepISOBuilder,
				Severity: host.SeverityOK,
				Detail:   "xorriso at /usr/bin/xorriso",
			},
			{
				Name:     host.DepSSH,
				Severity: host.SeverityOK,
				Detail:   "/usr/bin/ssh",
			},
			{
				Name:     host.DepKVM,
				Severity: host.SeverityWarning,
				Detail:   "present but not accessible by current user: permission denied",
			},
		}
	}

	service := NewService()
	service.discoverSSHKey = func() (string, error) { return "ssh-ed25519 AAAA", nil }
	service.resolveYeastHome = func() (string, error) { return "/home/test/.yeast", nil }

	result, err := service.Doctor()
	if err != nil {
		t.Fatalf("Doctor returned error: %v", err)
	}
	if result.Environment != string(host.EnvironmentContainer) {
		t.Fatalf("expected environment=%q, got %q", host.EnvironmentContainer, result.Environment)
	}

	var kvmCheck *DoctorCheck
	for i, c := range result.Checks {
		if c.Name == "/dev/kvm" {
			kvmCheck = &result.Checks[i]
			break
		}
	}
	if kvmCheck == nil {
		t.Fatal("expected /dev/kvm check")
	}
	if kvmCheck.Status != CheckStatusWarning {
		t.Fatalf("expected inaccessible KVM to stay a warning, got %s", kvmCheck.Status)
	}
	if !strings.Contains(kvmCheck.Details, "permission denied") {
		t.Fatalf("expected inaccessible KVM detail to mention permission denied, got %q", kvmCheck.Details)
	}
}

func TestDoctorISOBuilderAcceptsXorriso(t *testing.T) {
	previousLookPath := lookPath
	previousStatPath := statPath
	defer func() {
		lookPath = previousLookPath
		statPath = previousStatPath
	}()

	lookPath = func(file string) (string, error) {
		switch file {
		case "xorriso":
			return "/usr/bin/xorriso", nil
		case "qemu-system-x86_64", "qemu-img", "ssh":
			return "/usr/bin/" + file, nil
		}
		return "", errors.New("missing")
	}
	statPath = func(path string) (os.FileInfo, error) {
		if path == "/dev/kvm" {
			return fakeFileInfo{name: "kvm", mode: os.ModeDevice}, nil
		}
		if strings.HasSuffix(path, filepath.Join(".yeast", "cache", "images")) {
			return fakeFileInfo{name: "images", mode: os.ModeDir}, nil
		}
		return nil, os.ErrNotExist
	}

	service := NewService()
	service.discoverSSHKey = func() (string, error) { return "ssh-ed25519 AAAA", nil }
	service.resolveYeastHome = func() (string, error) { return "/home/test/.yeast", nil }

	result, err := service.Doctor()
	if err != nil {
		t.Fatalf("Doctor returned error: %v", err)
	}

	var isoCheck *DoctorCheck
	for i, c := range result.Checks {
		if c.Name == "iso-builder" {
			isoCheck = &result.Checks[i]
			break
		}
	}
	if isoCheck == nil {
		t.Fatal("expected iso-builder check")
	}
	if isoCheck.Status != CheckStatusOK {
		t.Fatalf("expected xorriso to satisfy iso-builder check, got %s: %s", isoCheck.Status, isoCheck.Details)
	}
}

func TestDoctorTreatsUnexpectedSSHKeyFailureAsWarning(t *testing.T) {
	previousLookPath := lookPath
	previousStatPath := statPath
	defer func() {
		lookPath = previousLookPath
		statPath = previousStatPath
	}()

	lookPath = func(file string) (string, error) {
		return "/usr/bin/" + file, nil
	}
	statPath = func(path string) (os.FileInfo, error) {
		if path == "/dev/kvm" {
			return fakeFileInfo{name: "kvm", mode: os.ModeDevice}, nil
		}
		return nil, os.ErrNotExist
	}

	service := NewService()
	service.discoverSSHKey = func() (string, error) { return "", errors.New("home lookup failed") }
	service.resolveYeastHome = func() (string, error) { return "/home/test/.yeast", nil }

	result, err := service.Doctor()
	if err != nil {
		t.Fatalf("Doctor returned error: %v", err)
	}

	var found bool
	for _, check := range result.Checks {
		if check.Name == "ssh-public-key" {
			found = true
			if check.Status != CheckStatusWarning {
				t.Fatalf("expected ssh-public-key warning, got %s", check.Status)
			}
		}
	}
	if !found {
		t.Fatal("expected ssh-public-key check")
	}
}

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
