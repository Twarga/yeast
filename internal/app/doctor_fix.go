package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type DoctorFixAction string

const (
	DoctorFixActionCommand        DoctorFixAction = "command"
	DoctorFixActionCreateSSHKey   DoctorFixAction = "create_ssh_key"
	DoctorFixActionCreateCacheDir DoctorFixAction = "create_cache_dir"
)

type DoctorFix struct {
	Action             DoctorFixAction `json:"action,omitempty"`
	Description        string          `json:"description,omitempty"`
	Command            []string        `json:"command,omitempty"`
	NeedsSudo          bool            `json:"needs_sudo,omitempty"`
	ManualOnly         bool            `json:"manual_only,omitempty"`
	ManualInstructions string          `json:"manual_instructions,omitempty"`
	Path               string          `json:"path,omitempty"`
}

type DoctorFixStep struct {
	CheckName string      `json:"check_name"`
	Status    CheckStatus `json:"status"`
	Fix       DoctorFix   `json:"fix"`
}

type DoctorFixPlan struct {
	Steps []DoctorFixStep `json:"steps"`
}

type DoctorFixApplyResult struct {
	Applied       []DoctorFixStep `json:"applied,omitempty"`
	SkippedManual []DoctorFixStep `json:"skipped_manual,omitempty"`
}

var (
	runDoctorCommand = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return exec.CommandContext(ctx, name, args...).CombinedOutput()
	}
	mkdirAll = os.MkdirAll
)

func BuildDoctorFixPlan(result DoctorResult) DoctorFixPlan {
	var plan DoctorFixPlan
	for _, check := range result.Checks {
		if check.Status == CheckStatusOK || check.Fix == nil {
			continue
		}
		plan.Steps = append(plan.Steps, DoctorFixStep{
			CheckName: check.Name,
			Status:    check.Status,
			Fix:       *check.Fix,
		})
	}
	return plan
}

func (p DoctorFixPlan) Empty() bool {
	return len(p.Steps) == 0
}

func (p DoctorFixPlan) AutomatableCount() int {
	count := 0
	for _, step := range p.Steps {
		if !step.Fix.ManualOnly {
			count++
		}
	}
	return count
}

func (p DoctorFixPlan) ManualOnlyCount() int {
	return len(p.Steps) - p.AutomatableCount()
}

func (p DoctorFixPlan) Describe() string {
	if p.Empty() {
		return "nothing to fix"
	}
	auto := p.AutomatableCount()
	manual := p.ManualOnlyCount()
	switch {
	case auto > 0 && manual > 0:
		return fmt.Sprintf("%d fix(es) can run automatically, %d require manual work", auto, manual)
	case auto > 0:
		return fmt.Sprintf("%d fix(es) can run automatically", auto)
	default:
		return fmt.Sprintf("%d fix(es) require manual work", manual)
	}
}

func (s *Service) ApplyDoctorFixes(ctx context.Context, plan DoctorFixPlan) (DoctorFixApplyResult, error) {
	var result DoctorFixApplyResult
	for _, step := range plan.Steps {
		if step.Fix.ManualOnly {
			result.SkippedManual = append(result.SkippedManual, step)
			continue
		}
		if err := applyDoctorFix(ctx, step); err != nil {
			return result, err
		}
		result.Applied = append(result.Applied, step)
	}
	return result, nil
}

func applyDoctorFix(ctx context.Context, step DoctorFixStep) error {
	switch step.Fix.Action {
	case DoctorFixActionCommand:
		return runDoctorCommandFix(ctx, step)
	case DoctorFixActionCreateSSHKey:
		return createDoctorSSHKey(ctx, step)
	case DoctorFixActionCreateCacheDir:
		return createDoctorCacheDir(step)
	default:
		return WrapError(ErrorCodeInternal, fmt.Sprintf("unsupported doctor fix action for %s", step.CheckName), nil)
	}
}

func runDoctorCommandFix(ctx context.Context, step DoctorFixStep) error {
	if len(step.Fix.Command) == 0 {
		return WrapError(ErrorCodeInternal, fmt.Sprintf("doctor fix for %s has no command", step.CheckName), nil)
	}

	cmdName := step.Fix.Command[0]
	args := append([]string{}, step.Fix.Command[1:]...)
	if step.Fix.NeedsSudo && os.Geteuid() != 0 {
		if _, err := exec.LookPath("sudo"); err != nil {
			return WrapError(ErrorCodePrecondition, "sudo is required to apply doctor fixes that install system dependencies", err)
		}
		args = append([]string{cmdName}, args...)
		cmdName = "sudo"
	}

	output, err := runDoctorCommand(ctx, cmdName, args...)
	if err != nil {
		commandLine := strings.Join(append([]string{cmdName}, args...), " ")
		err = WithDetails(WrapError(ErrorCodeRuntime, fmt.Sprintf("doctor fix failed while running: %s", commandLine), err), map[string]any{
			"command": commandLine,
			"output":  strings.TrimSpace(string(output)),
		})
		return err
	}
	return nil
}

func createDoctorSSHKey(ctx context.Context, step DoctorFixStep) error {
	if step.Fix.Path == "" {
		return WrapError(ErrorCodeInternal, "ssh key doctor fix is missing a target path", nil)
	}
	if err := mkdirAll(filepath.Dir(step.Fix.Path), 0o700); err != nil {
		return WrapError(ErrorCodeRuntime, fmt.Sprintf("failed to create %s", filepath.Dir(step.Fix.Path)), err)
	}
	output, err := runDoctorCommand(ctx, "ssh-keygen", "-t", "ed25519", "-f", step.Fix.Path, "-N", "")
	if err != nil {
		err = WithDetails(WrapError(ErrorCodeRuntime, fmt.Sprintf("failed to generate SSH key at %s", step.Fix.Path), err), map[string]any{
			"output": strings.TrimSpace(string(output)),
		})
		return err
	}
	return nil
}

func createDoctorCacheDir(step DoctorFixStep) error {
	if step.Fix.Path == "" {
		return WrapError(ErrorCodeInternal, "cache-directory doctor fix is missing a target path", nil)
	}
	if err := mkdirAll(step.Fix.Path, 0o755); err != nil {
		return WrapError(ErrorCodeRuntime, fmt.Sprintf("failed to create cache directory %s", step.Fix.Path), err)
	}
	return nil
}
