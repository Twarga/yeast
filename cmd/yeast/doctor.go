package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"slices"

	"github.com/spf13/cobra"
)

type doctorLevel int

const (
	levelOK doctorLevel = iota
	levelWarning
	levelBlocker
)

type doctorResult struct {
	Name    string
	Level   doctorLevel
	Message string
	Fixes   []string
}

func (l doctorLevel) value() string {
	switch l {
	case levelOK:
		return "ok"
	case levelWarning:
		return "warning"
	case levelBlocker:
		return "blocker"
	default:
		return "unknown"
	}
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Run environment preflight checks",
	Long:  "Checks host prerequisites needed by Yeast and prints remediation steps.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !outputJSON {
			humanSection("Running Preflight Checks")
			fmt.Println()
		}

		results := []doctorResult{
			checkBinary("qemu-system-x86_64", "QEMU system emulator"),
			checkBinary("qemu-img", "QEMU image tooling"),
			checkBinary("genisoimage", "Cloud-init ISO builder"),
		}

		kvmResult, kvmAccessOK := checkKVMDevice()
		results = append(results, kvmResult)
		results = append(results, checkKVMGroupMembership(kvmAccessOK))
		results = append(results, checkSSHKeys())
		results = append(results, checkCacheDir())

		var warnings int
		var blockers int
		jsonChecks := make([]doctorCheckOutput, 0, len(results))
		for _, r := range results {
			if !outputJSON {
				switch r.Level {
				case levelOK:
					humanSuccessf("%s: %s", r.Name, r.Message)
				case levelWarning:
					humanWarnf("%s: %s", r.Name, r.Message)
				case levelBlocker:
					humanErrorf("%s: %s", r.Name, r.Message)
				default:
					humanInfof("%s: %s", r.Name, r.Message)
				}
				for _, fix := range r.Fixes {
					humanKeyValue("Fix", fix)
				}
			}
			jsonChecks = append(jsonChecks, doctorCheckOutput{
				Name:    r.Name,
				Level:   r.Level.value(),
				Message: r.Message,
				Fixes:   r.Fixes,
			})

			switch r.Level {
			case levelWarning:
				warnings++
			case levelBlocker:
				blockers++
			}
		}

		jsonData := doctorCommandData{
			Schema:   "yeast.doctor.v1",
			Checks:   jsonChecks,
			Total:    len(results),
			Blockers: blockers,
			Warnings: warnings,
		}

		if outputJSON {
			if blockers > 0 {
				return jsonCommandErrorWithData("doctor", "preflight_blockers", fmt.Errorf("preflight failed with %d blocker(s)", blockers), jsonData)
			}
			return jsonCommandSuccess("doctor", jsonData)
		}

		fmt.Println()
		humanSection("Preflight Summary")
		humanKeyValue("Checks", fmt.Sprintf("%d", len(results)))
		humanKeyValue("Blockers", fmt.Sprintf("%d", blockers))
		humanKeyValue("Warnings", fmt.Sprintf("%d", warnings))
		if blockers > 0 {
			return fmt.Errorf("preflight failed with %d blocker(s)", blockers)
		}

		if warnings > 0 {
			fmt.Println()
			humanWarnf("Preflight passed with warnings")
			return nil
		}

		fmt.Println()
		humanSuccessf("Preflight passed. Environment looks good")
		return nil
	},
}

func checkBinary(bin, description string) doctorResult {
	path, err := exec.LookPath(bin)
	if err == nil {
		return doctorResult{
			Name:    description,
			Level:   levelOK,
			Message: fmt.Sprintf("found at %s", path),
		}
	}

	return doctorResult{
		Name:    description,
		Level:   levelBlocker,
		Message: fmt.Sprintf("%s was not found in PATH", bin),
		Fixes: []string{
			"Ubuntu/Debian: sudo apt install qemu-system-x86 qemu-utils genisoimage",
			"Fedora/RHEL: sudo dnf install qemu-system-x86 qemu-img genisoimage",
			"Arch: sudo pacman -S qemu-base cdrtools",
		},
	}
}

func checkKVMDevice() (doctorResult, bool) {
	info, err := os.Stat("/dev/kvm")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return doctorResult{
				Name:    "KVM device access",
				Level:   levelBlocker,
				Message: "/dev/kvm does not exist",
				Fixes: []string{
					"Enable hardware virtualization (VT-x/AMD-V) in BIOS/UEFI.",
					"Install KVM packages (qemu-system-x86 and related virtualization packages).",
				},
			}, false
		}
		return doctorResult{
			Name:    "KVM device access",
			Level:   levelBlocker,
			Message: fmt.Sprintf("failed to stat /dev/kvm: %v", err),
			Fixes: []string{
				"Verify host virtualization setup and permissions.",
			},
		}, false
	}

	if info.Mode()&os.ModeDevice == 0 {
		return doctorResult{
			Name:    "KVM device access",
			Level:   levelBlocker,
			Message: "/dev/kvm exists but is not a device node",
			Fixes: []string{
				"Check host KVM kernel modules and virtualization setup.",
			},
		}, false
	}

	f, err := os.OpenFile("/dev/kvm", os.O_RDWR, 0)
	if err != nil {
		if os.IsPermission(err) {
			return doctorResult{
				Name:    "KVM device access",
				Level:   levelBlocker,
				Message: "permission denied when opening /dev/kvm",
				Fixes: []string{
					"Add your user to the kvm group: sudo usermod -aG kvm $USER",
					"Log out and log back in to refresh group membership.",
				},
			}, false
		}
		return doctorResult{
			Name:    "KVM device access",
			Level:   levelBlocker,
			Message: fmt.Sprintf("cannot open /dev/kvm: %v", err),
			Fixes: []string{
				"Verify KVM permissions and that no host policy blocks device access.",
			},
		}, false
	}
	_ = f.Close()

	return doctorResult{
		Name:    "KVM device access",
		Level:   levelOK,
		Message: "/dev/kvm is present and accessible",
	}, true
}

func checkKVMGroupMembership(kvmAccessOK bool) doctorResult {
	if os.Geteuid() == 0 {
		return doctorResult{
			Name:    "kvm group membership",
			Level:   levelWarning,
			Message: "running as root; group membership check skipped",
			Fixes: []string{
				"Prefer running Yeast as a non-root user in the kvm group.",
			},
		}
	}

	currentUser, err := user.Current()
	if err != nil {
		return doctorResult{
			Name:    "kvm group membership",
			Level:   levelWarning,
			Message: fmt.Sprintf("could not resolve current user: %v", err),
		}
	}

	group, err := user.LookupGroup("kvm")
	if err != nil {
		return doctorResult{
			Name:    "kvm group membership",
			Level:   levelWarning,
			Message: "kvm group does not exist on this host",
			Fixes: []string{
				"Install host virtualization packages that create the kvm group.",
			},
		}
	}

	gids, err := currentUser.GroupIds()
	if err != nil {
		return doctorResult{
			Name:    "kvm group membership",
			Level:   levelWarning,
			Message: fmt.Sprintf("could not inspect user groups: %v", err),
		}
	}

	if slices.Contains(gids, group.Gid) {
		return doctorResult{
			Name:    "kvm group membership",
			Level:   levelOK,
			Message: fmt.Sprintf("user %s is in kvm group", currentUser.Username),
		}
	}

	level := levelWarning
	if !kvmAccessOK {
		level = levelBlocker
	}
	return doctorResult{
		Name:    "kvm group membership",
		Level:   level,
		Message: fmt.Sprintf("user %s is not in kvm group", currentUser.Username),
		Fixes: []string{
			"Run: sudo usermod -aG kvm $USER",
			"Log out and log back in, then re-run `yeast doctor`.",
		},
	}
}

func checkSSHKeys() doctorResult {
	home, err := os.UserHomeDir()
	if err != nil {
		return doctorResult{
			Name:    "SSH key presence",
			Level:   levelBlocker,
			Message: fmt.Sprintf("cannot resolve user home directory: %v", err),
		}
	}

	candidates := []string{
		filepath.Join(home, ".ssh", "id_ed25519.pub"),
		filepath.Join(home, ".ssh", "id_rsa.pub"),
	}
	for _, keyPath := range candidates {
		if _, err := os.Stat(keyPath); err == nil {
			return doctorResult{
				Name:    "SSH key presence",
				Level:   levelOK,
				Message: fmt.Sprintf("found public key: %s", keyPath),
			}
		}
	}

	return doctorResult{
		Name:    "SSH key presence",
		Level:   levelBlocker,
		Message: "no supported SSH public key found (~/.ssh/id_ed25519.pub or ~/.ssh/id_rsa.pub)",
		Fixes: []string{
			"Generate one: ssh-keygen -t ed25519 -N \"\" -C \"yeast@localhost\"",
		},
	}
}

func checkCacheDir() doctorResult {
	home, err := os.UserHomeDir()
	if err != nil {
		return doctorResult{
			Name:    "Image cache directory",
			Level:   levelWarning,
			Message: fmt.Sprintf("cannot resolve user home directory: %v", err),
		}
	}

	cacheDir := filepath.Join(home, ".yeast", "cache")
	info, err := os.Stat(cacheDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return doctorResult{
				Name:    "Image cache directory",
				Level:   levelWarning,
				Message: fmt.Sprintf("%s does not exist", cacheDir),
				Fixes: []string{
					fmt.Sprintf("Create it: mkdir -p %s", cacheDir),
					"Place cloud images there (e.g. ubuntu-22.04.img).",
				},
			}
		}
		return doctorResult{
			Name:    "Image cache directory",
			Level:   levelWarning,
			Message: fmt.Sprintf("cannot read %s: %v", cacheDir, err),
		}
	}

	if !info.IsDir() {
		return doctorResult{
			Name:    "Image cache directory",
			Level:   levelBlocker,
			Message: fmt.Sprintf("%s exists but is not a directory", cacheDir),
			Fixes: []string{
				fmt.Sprintf("Remove or rename it and create directory: mkdir -p %s", cacheDir),
			},
		}
	}

	testFile := filepath.Join(cacheDir, ".yeast-doctor-write-test")
	if err := os.WriteFile(testFile, []byte("ok"), 0600); err != nil {
		return doctorResult{
			Name:    "Image cache directory",
			Level:   levelWarning,
			Message: fmt.Sprintf("%s is not writable: %v", cacheDir, err),
			Fixes: []string{
				fmt.Sprintf("Fix permissions: chmod u+rwx %s", cacheDir),
			},
		}
	}
	_ = os.Remove(testFile)

	images, _ := filepath.Glob(filepath.Join(cacheDir, "*.img"))
	if len(images) == 0 {
		return doctorResult{
			Name:    "Image cache directory",
			Level:   levelWarning,
			Message: "cache directory is empty (no *.img base image found)",
			Fixes: []string{
				"Download and verify a trusted image: yeast pull ubuntu-22.04",
			},
		}
	}

	return doctorResult{
		Name:    "Image cache directory",
		Level:   levelOK,
		Message: fmt.Sprintf("found %d image file(s) in cache", len(images)),
	}
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
