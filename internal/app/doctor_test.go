package app

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"yeast/internal/provision/cloudinit"
)

func TestDoctorReportsHealthyHost(t *testing.T) {
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
	defer func() {
		lookPath = previousLookPath
		statPath = previousStatPath
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
