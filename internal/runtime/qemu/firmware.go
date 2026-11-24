package qemu

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ovmfCandidates lists (CODE, VARS) pairs across common distros:
// Ubuntu/Debian (4M and legacy), Fedora/RHEL, Arch.
var ovmfCandidates = [][2]string{
	{"/usr/share/OVMF/OVMF_CODE_4M.fd", "/usr/share/OVMF/OVMF_VARS_4M.fd"},
	{"/usr/share/OVMF/OVMF_CODE.fd", "/usr/share/OVMF/OVMF_VARS.fd"},
	{"/usr/share/edk2/ovmf/OVMF_CODE.fd", "/usr/share/edk2/ovmf/OVMF_VARS.fd"},
	{"/usr/share/edk2/ovmf/OVMF_CODE_4M.secboot.fd", "/usr/share/edk2/ovmf/OVMF_VARS_4M.fd"},
	{"/usr/share/edk2-ovmf/x64/OVMF_CODE.fd", "/usr/share/edk2-ovmf/x64/OVMF_VARS.fd"},
	{"/usr/share/qemu/OVMF_CODE.fd", "/usr/share/qemu/OVMF_VARS.fd"},
}

// overridable for tests
var statFileFn = os.Stat
var copyFileFn = copyFileContents

// ovmfVarsName is the per-VM EFI variable store inside the runtime dir.
const ovmfVarsName = "ovmf_vars.fd"

// DiscoverFirmware finds an OVMF (CODE, VARS) pair on the host. It returns a
// clear actionable error when no pair exists.
func DiscoverFirmware() (code, vars string, err error) {
	for _, pair := range ovmfCandidates {
		if _, err := statFileFn(pair[0]); err != nil {
			continue
		}
		if _, err := statFileFn(pair[1]); err != nil {
			continue
		}
		return pair[0], pair[1], nil
	}
	return "", "", fmt.Errorf("no OVMF UEFI firmware found on this host; install the ovmf package (Debian/Ubuntu: sudo apt install ovmf, Fedora: sudo dnf install edk2-ovmf, Arch: sudo pacman -S edk2-ovmf) or use firmware: bios")
}

// discoverFirmwareFn allows tests to stub discovery.
var discoverFirmwareFn = DiscoverFirmware

// prepareFirmware copies OVMF_VARS into the runtime dir so the VM owns its
// EFI variables persistently. Idempotent: an existing vars file is kept.
func prepareFirmware(runtimeDir string) (codePath, varsPath string, err error) {
	code, vars, err := discoverFirmwareFn()
	if err != nil {
		return "", "", err
	}
	varsPath = filepath.Join(runtimeDir, ovmfVarsName)
	if _, err := statFileFn(varsPath); err == nil {
		return code, varsPath, nil
	}
	if err := copyFileFn(vars, varsPath, 0o644); err != nil {
		return "", "", fmt.Errorf("copy OVMF_VARS to %s: %w", varsPath, err)
	}
	return code, varsPath, nil
}

// prepareFirmwareFn is injectable for command tests.
var prepareFirmwareFn = prepareFirmware

func copyFileContents(src, dst string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}
