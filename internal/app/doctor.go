package app

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"yeast/internal/host"
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
	Fix     *DoctorFix  `json:"fix,omitempty"`
}

type DoctorResult struct {
	Checks      []DoctorCheck `json:"checks"`
	Blockers    int           `json:"blockers"`
	Warnings    int           `json:"warnings"`
	Environment string        `json:"environment,omitempty"` // e.g. "native-linux", "wsl2"
	SupportTier string        `json:"support_tier,omitempty"` // "A", "B", "C", "D"
}

var (
	lookPath              = exec.LookPath
	statPath              = os.Stat
	detectHostEnvironment = host.Detect
	runHostChecks         = host.RunChecks
	userHomeDir           = os.UserHomeDir
)

func (s *Service) Doctor() (DoctorResult, error) {
	env := detectHostEnvironment()

	result := DoctorResult{
		Checks:      make([]DoctorCheck, 0, 7),
		Environment: string(env.Type),
		SupportTier: string(env.SupportTier),
	}

	for _, check := range runHostChecks(env, lookPath, statPath) {
		result.addCheck(doctorCheckFromHost(check))
	}
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

func doctorCheckFromHost(check host.CheckResult) DoctorCheck {
	return DoctorCheck{
		Name:    string(check.Name),
		Status:  doctorStatusFromHost(check.Severity),
		Details: check.Detail,
		Fix:     doctorFixFromHost(check.Fix),
	}
}

func doctorStatusFromHost(severity host.CheckSeverity) CheckStatus {
	switch severity {
	case host.SeverityOK:
		return CheckStatusOK
	case host.SeverityWarning:
		return CheckStatusWarning
	default:
		return CheckStatusBlocker
	}
}

func doctorFixFromHost(fix *host.FixStep) *DoctorFix {
	if fix == nil {
		return nil
	}
	command := append([]string{}, fix.Command...)
	return &DoctorFix{
		Action:             DoctorFixActionCommand,
		Description:        fix.Description,
		Command:            command,
		NeedsSudo:          fix.NeedsSudo,
		ManualOnly:         fix.ManualOnly,
		ManualInstructions: fix.ManualInstructions,
	}
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
	var fix *DoctorFix
	if !errors.Is(err, cloudinit.ErrNoSSHPublicKey) {
		status = CheckStatusWarning
	} else if home, homeErr := userHomeDir(); homeErr == nil {
		fix = &DoctorFix{
			Action:      DoctorFixActionCreateSSHKey,
			Description: "generate a default ed25519 SSH key for Yeast guest access",
			Path:        filepath.Join(home, ".ssh", "id_ed25519"),
		}
	}
	r.addCheck(DoctorCheck{
		Name:    "ssh-public-key",
		Status:  status,
		Details: err.Error(),
		Fix:     fix,
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
			Fix: &DoctorFix{
				Description:        "replace the cache path with a directory",
				ManualOnly:         true,
				ManualInstructions: fmt.Sprintf("Move or delete %s, then rerun `yeast doctor --fix --yes`.", cacheDir),
			},
		})
		return
	}

	r.addCheck(DoctorCheck{
		Name:    "cache-directory",
		Status:  CheckStatusWarning,
		Details: fmt.Sprintf("%s does not exist yet; it will be created on first image pull", cacheDir),
		Fix: &DoctorFix{
			Action:      DoctorFixActionCreateCacheDir,
			Description: "create the Yeast image cache directory",
			Path:        cacheDir,
		},
	})
}
