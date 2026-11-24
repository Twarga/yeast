package qemu

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	rtm "github.com/Twarga/yeast/internal/runtime"
)

func TestDiscoverFirmwarePairs(t *testing.T) {
	defer func(f func(string) (os.FileInfo, error)) { statFileFn = f }(statFileFn)

	statFileFn = func(string) (os.FileInfo, error) {
		return nil, errors.New("missing")
	}
	if _, _, err := discoverFirmwareFn(); err == nil {
		t.Fatal("expected missing-OVMF error")
	} else if !strings.Contains(err.Error(), "OVMF") {
		t.Fatalf("error should mention OVMF: %v", err)
	}

	statFileFn = func(path string) (os.FileInfo, error) {
		if path == "/usr/share/edk2/ovmf/OVMF_CODE.fd" || path == "/usr/share/edk2/ovmf/OVMF_VARS.fd" {
			return nil, nil
		}
		return nil, errors.New("missing")
	}
	code, vars, err := discoverFirmwareFn()
	if err != nil {
		t.Fatal(err)
	}
	if code != "/usr/share/edk2/ovmf/OVMF_CODE.fd" || vars != "/usr/share/edk2/ovmf/OVMF_VARS.fd" {
		t.Fatalf("code=%q vars=%q", code, vars)
	}
}

func TestPrepareFirmwareCopiesVarsOnce(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "OVMF_VARS.fd")
	if err := os.WriteFile(src, []byte("VARS-BYTES"), 0o644); err != nil {
		t.Fatal(err)
	}

	defer func(f func() (string, string, error)) { discoverFirmwareFn = f }(discoverFirmwareFn)
	discoverFirmwareFn = func() (string, string, error) {
		return filepath.Join(dir, "OVMF_CODE.fd"), src, nil
	}
	defer func(f func(string) (os.FileInfo, error)) { statFileFn = f }(statFileFn)
	statFileFn = os.Stat

	runtimeDir := filepath.Join(dir, "rt")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	code, varsPath, err := prepareFirmwareFn(runtimeDir)
	if err != nil {
		t.Fatal(err)
	}
	if varsPath != filepath.Join(runtimeDir, ovmfVarsName) {
		t.Fatalf("varsPath = %q", varsPath)
	}
	data, err := os.ReadFile(varsPath)
	if err != nil || string(data) != "VARS-BYTES" {
		t.Fatalf("vars copy: %q err=%v", string(data), err)
	}

	// second call keeps an existing vars file
	if err := os.WriteFile(varsPath, []byte("MODIFIED"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := prepareFirmwareFn(runtimeDir); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(varsPath)
	if string(data) != "MODIFIED" {
		t.Fatal("existing ovmf_vars.fd was overwritten")
	}
	_ = code
}

func TestPrepareDiskUEFICopiesVars(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "OVMF_VARS.fd")
	if err := os.WriteFile(src, []byte("V"), 0o644); err != nil {
		t.Fatal(err)
	}
	defer func(f func() (string, string, error)) { discoverFirmwareFn = f }(discoverFirmwareFn)
	discoverFirmwareFn = func() (string, string, error) {
		return filepath.Join(dir, "CODE.fd"), src, nil
	}
	defer func(f func(string) (os.FileInfo, error)) { statFileFn = f }(statFileFn)
	statFileFn = os.Stat
	defer func(f func(ctx context.Context, name string, args ...string) error) { runCommand = f }(runCommand)
	runCommand = func(context.Context, string, ...string) error { return nil }

	plan := basePlan()
	plan.Firmware = "uefi"
	plan.RuntimeDir = filepath.Join(dir, "rt")
	plan.Disk = rtmDiskPlan(dir)
	if _, err := PrepareDisk(context.Background(), plan); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(plan.RuntimeDir, ovmfVarsName)); err != nil {
		t.Fatalf("ovmf_vars.fd not prepared: %v", err)
	}
}

func rtmDiskPlan(dir string) rtm.DiskPlan {
	base := filepath.Join(dir, "base.qcow2")
	_ = os.WriteFile(base, []byte("x"), 0o644)
	return rtm.DiskPlan{BaseImagePath: base, DiskPath: filepath.Join(dir, "rt", "disk.qcow2")}
}
