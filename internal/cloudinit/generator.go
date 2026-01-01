package cloudinit

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"yeast/internal/types"
)

// GenerateSeedISO creates a cloud-init seed.iso containing user-data and meta-data
func GenerateSeedISO(machine types.Machine, outputPath string) error {
	tmpDir, err := os.MkdirTemp("", "yeast-cloudinit-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	userData, err := generateUserData(machine)
	if err != nil {
		return err
	}

	metaData := generateMetaData(machine)

	if err := os.WriteFile(filepath.Join(tmpDir, "user-data"), []byte(userData), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "meta-data"), []byte(metaData), 0644); err != nil {
		return err
	}

	// Create ISO using xorriso (mimicking mkisofs)
	cmd := exec.Command("xorriso", "-as", "mkisofs", "-R", "-V", "config-2", "-o", outputPath, tmpDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("xorriso failed: %w, output: %s", err, string(output))
	}

	return nil
}

func generateUserData(machine types.Machine) (string, error) {
	// For MVP, if a template path is provided, we read it. 
	// Otherwise, we provide a very basic default.
	if machine.Template != "" {
		content, err := os.ReadFile(machine.Template)
		if err != nil {
			return "", fmt.Errorf("failed to read template %s: %w", machine.Template, err)
		}
		return string(content), nil
	}

	// Minimal default user-data
	return "#cloud-config\nusers:\n  - name: yeast\n    sudo: ALL=(ALL) NOPASSWD:ALL\n    shell: /bin/bash\nssh_pwauth: True\nchpasswd:\n  list: |\n     yeast:yeast\n  expire: False\n", nil
}

func generateMetaData(machine types.Machine) string {
	return fmt.Sprintf("instance-id: %s\nlocal-hostname: %s\n", machine.Name, machine.Name)
}
