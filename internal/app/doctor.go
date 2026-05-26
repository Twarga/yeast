package app

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"yeast/internal/project"
	"yeast/internal/provision/cloudinit"
)

type CheckStatus string

const (
	CheckStatusOK      CheckStatus = "ok"
	CheckStatusWarning CheckStatus = "warning"
	CheckStatusBlocker CheckStatus = "blocker"
)

type DoctorCheck struct {
	Name    string      `json:"name"`
	Status  CheckStatus `json:"status"`
	Details string      `json:"details"`
}

type DoctorResult struct {
	Checks   []DoctorCheck `json:"checks"`
	Blockers int           `json:"blockers"`
	Warnings int           `json:"warnings"`
}

var (
	lookPath = exec.LookPath
	statPath = os.Stat
)

func (s *Service) Doctor() (DoctorResult, error) {
	result := DoctorResult{
		Checks: make([]DoctorCheck, 0, 7),
	}

	result.addBinaryCheck("qemu-system-x86_64", "required to start virtual machines")
	result.addBinaryCheck("qemu-img", "required to create qcow2 overlay disks")
	result.addISOBuilderCheck()
	result.addBinaryCheck("ssh", "required for yeast ssh and guest control")
	result.addKVMCheck()
	result.addSSHKeyCheck(s.discoverSSHKey)
	result.addCacheDirCheck(s.resolveYeastHome)

	return result, nil
}

func (r *DoctorResult) addCheck(check DoctorCheck) {
	r.Checks = append(r.Checks, check)
	switch check.Status {
	case CheckStatusBlocker:
		r.Blockers++
	case CheckStatusWarning:
		r.Warnings++
	}
}

func (r *DoctorResult) addBinaryCheck(name, purpose string) {
	if path, err := lookPath(name); err == nil {
		r.addCheck(DoctorCheck{
			Name:    name,
			Status:  CheckStatusOK,
			Details: path,
		})
		return
	}
	r.addCheck(DoctorCheck{
		Name:    name,
		Status:  CheckStatusBlocker,
		Details: purpose,
	})
}

func (r *DoctorResult) addISOBuilderCheck() {
	for _, name := range []string{"genisoimage", "mkisofs"} {
		if path, err := lookPath(name); err == nil {
			r.addCheck(DoctorCheck{
				Name:    "iso-builder",
				Status:  CheckStatusOK,
				Details: fmt.Sprintf("%s at %s", name, path),
			})
			return
		}
	}

	r.addCheck(DoctorCheck{
		Name:    "iso-builder",
		Status:  CheckStatusBlocker,
		Details: "install genisoimage or mkisofs to build cloud-init seed ISOs",
	})
}

func (r *DoctorResult) addKVMCheck() {
	info, err := statPath("/dev/kvm")
	if err != nil {
		r.addCheck(DoctorCheck{
			Name:    "/dev/kvm",
			Status:  CheckStatusBlocker,
			Details: "missing or inaccessible; KVM acceleration is required",
		})
		return
	}
	mode := info.Mode()
	if mode&os.ModeDevice == 0 {
		r.addCheck(DoctorCheck{
			Name:    "/dev/kvm",
			Status:  CheckStatusBlocker,
			Details: "exists but is not a device",
		})
		return
	}
	r.addCheck(DoctorCheck{
		Name:    "/dev/kvm",
		Status:  CheckStatusOK,
		Details: "present",
	})
}

func (r *DoctorResult) addSSHKeyCheck(discover func() (string, error)) {
	key, err := discover()
	if err == nil {
		r.addCheck(DoctorCheck{
			Name:    "ssh-public-key",
			Status:  CheckStatusOK,
			Details: fmt.Sprintf("%d bytes", len(key)),
		})
		return
	}

	status := CheckStatusBlocker
	if !errors.Is(err, cloudinit.ErrNoSSHPublicKey) {
		status = CheckStatusWarning
	}
	r.addCheck(DoctorCheck{
		Name:    "ssh-public-key",
		Status:  status,
		Details: err.Error(),
	})
}

func (r *DoctorResult) addCacheDirCheck(resolveYeastHome func() (string, error)) {
	home, err := resolveYeastHome()
	if err != nil {
		r.addCheck(DoctorCheck{
			Name:    "cache-directory",
			Status:  CheckStatusWarning,
			Details: err.Error(),
		})
		return
	}

	cacheDir := filepath.Join(home, project.CacheDirName, project.ImagesDirName)
	if info, err := statPath(cacheDir); err == nil {
		if info.IsDir() {
			r.addCheck(DoctorCheck{
				Name:    "cache-directory",
				Status:  CheckStatusOK,
				Details: cacheDir,
			})
			return
		}
		r.addCheck(DoctorCheck{
			Name:    "cache-directory",
			Status:  CheckStatusWarning,
			Details: fmt.Sprintf("%s exists but is not a directory", cacheDir),
		})
		return
	}

	r.addCheck(DoctorCheck{
		Name:    "cache-directory",
		Status:  CheckStatusWarning,
		Details: fmt.Sprintf("%s does not exist yet; it will be created on first image pull", cacheDir),
	})
}
