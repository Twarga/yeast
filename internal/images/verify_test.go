package images

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifySHA256CorrectChecksum(t *testing.T) {
	path := filepath.Join(t.TempDir(), "image.qcow2")
	if err := os.WriteFile(path, []byte("hello"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	err := VerifySHA256(path, "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824")
	if err != nil {
		t.Fatalf("expected checksum verification to pass, got %v", err)
	}
}

func TestVerifySHA256WrongChecksum(t *testing.T) {
	path := filepath.Join(t.TempDir(), "image.qcow2")
	if err := os.WriteFile(path, []byte("hello"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	err := VerifySHA256(path, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err == nil {
		t.Fatal("expected checksum mismatch error")
	}
	if !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("expected checksum mismatch error, got %v", err)
	}
}

func TestVerifySHA256MissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.qcow2")

	err := VerifySHA256(path, "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824")
	if err == nil {
		t.Fatal("expected missing file error")
	}
	if !strings.Contains(err.Error(), "open file for checksum") {
		t.Fatalf("expected open file for checksum error, got %v", err)
	}
}
