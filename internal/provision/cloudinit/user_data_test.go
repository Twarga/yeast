package cloudinit

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverAuthorizedKeyPrefersEd25519(t *testing.T) {
	home := t.TempDir()
	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0755); err != nil {
		t.Fatalf("create ssh dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sshDir, "id_ed25519.pub"), []byte("ssh-ed25519 AAAATEST-ED25519\n"), 0644); err != nil {
		t.Fatalf("write ed25519 key: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sshDir, "id_rsa.pub"), []byte("ssh-rsa AAAATEST-RSA\n"), 0644); err != nil {
		t.Fatalf("write rsa key: %v", err)
	}

	t.Setenv("HOME", home)

	key, err := DiscoverAuthorizedKey()
	if err != nil {
		t.Fatalf("DiscoverAuthorizedKey returned error: %v", err)
	}
	if key != "ssh-ed25519 AAAATEST-ED25519" {
		t.Fatalf("unexpected key: got %q", key)
	}
}

func TestDiscoverAuthorizedKeyFallsBackToRSA(t *testing.T) {
	home := t.TempDir()
	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0755); err != nil {
		t.Fatalf("create ssh dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sshDir, "id_rsa.pub"), []byte("ssh-rsa AAAATEST-RSA\n"), 0644); err != nil {
		t.Fatalf("write rsa key: %v", err)
	}

	t.Setenv("HOME", home)

	key, err := DiscoverAuthorizedKey()
	if err != nil {
		t.Fatalf("DiscoverAuthorizedKey returned error: %v", err)
	}
	if key != "ssh-rsa AAAATEST-RSA" {
		t.Fatalf("unexpected key: got %q", key)
	}
}

func TestDiscoverAuthorizedKeyMissing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	_, err := DiscoverAuthorizedKey()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrNoSSHPublicKey) {
		t.Fatalf("expected ErrNoSSHPublicKey, got %v", err)
	}
	if !strings.Contains(err.Error(), "id_ed25519.pub") || !strings.Contains(err.Error(), "id_rsa.pub") {
		t.Fatalf("expected candidate paths in error, got %q", err)
	}
}

func TestDiscoverAuthorizedKeyRejectsEmptyKey(t *testing.T) {
	home := t.TempDir()
	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0755); err != nil {
		t.Fatalf("create ssh dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sshDir, "id_ed25519.pub"), []byte("\n"), 0644); err != nil {
		t.Fatalf("write empty key: %v", err)
	}

	t.Setenv("HOME", home)

	_, err := DiscoverAuthorizedKey()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "is empty") {
		t.Fatalf("expected empty-key error, got %q", err)
	}
}
