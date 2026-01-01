package cloudinit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"yeast/internal/types"
)

func TestGenerateUserData_Default(t *testing.T) {
	m := types.Machine{Name: "test"}
	got, err := generateUserData(m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "#cloud-config") {
		t.Errorf("expected #cloud-config header, got %s", got)
	}
}

func TestGenerateUserData_Template(t *testing.T) {
	tmpDir := t.TempDir()
	templatePath := filepath.Join(tmpDir, "template.yaml")
	content := "custom: template"
	os.WriteFile(templatePath, []byte(content), 0644)

	m := types.Machine{Name: "test", Template: templatePath}
	got, err := generateUserData(m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != content {
		t.Errorf("expected %s, got %s", content, got)
	}
}

func TestGenerateMetaData(t *testing.T) {
	m := types.Machine{Name: "my-vm"}
	got := generateMetaData(m)
	if !strings.Contains(got, "instance-id: my-vm") {
		t.Errorf("expected instance-id: my-vm, got %s", got)
	}
	if !strings.Contains(got, "local-hostname: my-vm") {
		t.Errorf("expected local-hostname: my-vm, got %s", got)
	}
}

func TestGenerateSeedISO(t *testing.T) {
	tmpDir := t.TempDir()
	isoPath := filepath.Join(tmpDir, "seed.iso")
	m := types.Machine{Name: "test-iso"}

	err := GenerateSeedISO(m, isoPath)
	if err != nil {
		t.Fatalf("GenerateSeedISO failed: %v", err)
	}

	if _, err := os.Stat(isoPath); os.IsNotExist(err) {
		t.Errorf("ISO file was not created at %s", isoPath)
	}
}
