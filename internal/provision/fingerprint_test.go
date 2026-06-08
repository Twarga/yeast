package provision

import (
	"os"
	"path/filepath"
	"testing"

	"yeast/internal/config"
)

func TestProvisionFingerprintChangesWhenFileContentChanges(t *testing.T) {
	root := t.TempDir()

	file1 := filepath.Join(root, "file1.txt")
	if err := os.WriteFile(file1, []byte("content-a"), 0644); err != nil {
		t.Fatalf("write file1: %v", err)
	}
	file2 := filepath.Join(root, "file2.txt")
	if err := os.WriteFile(file2, []byte("content-b"), 0644); err != nil {
		t.Fatalf("write file2: %v", err)
	}

	instance := config.Instance{
		Name: "web",
		Provision: &config.ProvisionConfig{
			Files: []config.FileProvision{
				{Source: file1, Destination: "/tmp/file.txt"},
			},
		},
	}
	fp1, err := Fingerprint(root, instance, &config.Config{})
	if err != nil {
		t.Fatalf("fingerprint 1: %v", err)
	}

	instance.Provision.Files[0].Source = file2
	fp2, err := Fingerprint(root, instance, &config.Config{})
	if err != nil {
		t.Fatalf("fingerprint 2: %v", err)
	}

	if fp1 == fp2 {
		t.Fatalf("expected different fingerprints for different file contents, got %q and %q", fp1, fp2)
	}
}

func TestProvisionFingerprintPreservesShellOrder(t *testing.T) {
	root := t.TempDir()

	instance1 := config.Instance{
		Name: "web",
		Provision: &config.ProvisionConfig{
			Shell: []string{"echo first", "echo second"},
		},
	}
	instance2 := config.Instance{
		Name: "web",
		Provision: &config.ProvisionConfig{
			Shell: []string{"echo second", "echo first"},
		},
	}

	fp1, err := Fingerprint(root, instance1, &config.Config{})
	if err != nil {
		t.Fatalf("fingerprint 1: %v", err)
	}
	fp2, err := Fingerprint(root, instance2, &config.Config{})
	if err != nil {
		t.Fatalf("fingerprint 2: %v", err)
	}

	if fp1 == fp2 {
		t.Fatalf("expected different fingerprints for different shell order, got %q and %q", fp1, fp2)
	}
}

func TestProvisionFingerprintEmptyPlanIsStable(t *testing.T) {
	root := t.TempDir()

	instance := config.Instance{Name: "web"}
	fp1, err := Fingerprint(root, instance, &config.Config{})
	if err != nil {
		t.Fatalf("fingerprint 1: %v", err)
	}
	fp2, err := Fingerprint(root, instance, &config.Config{})
	if err != nil {
		t.Fatalf("fingerprint 2: %v", err)
	}

	if fp1 != fp2 {
		t.Fatalf("expected same fingerprint for empty plan, got %q and %q", fp1, fp2)
	}
}
