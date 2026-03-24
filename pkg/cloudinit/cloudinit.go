package cloudinit

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type UserData struct {
	Hostname   string
	SSHKey     string
	Username   string
	SudoPolicy SudoPolicy
	Env        map[string]string
}

type SudoPolicy string

const (
	SudoNone     SudoPolicy = "none"
	SudoPassword SudoPolicy = "password"
	SudoNoPass   SudoPolicy = "nopasswd"
)

func GenerateUserData(ud UserData) string {
	username := strings.TrimSpace(ud.Username)
	if username == "" {
		username = "yeast"
	}

	sudoPolicy := normalizeSudoPolicy(ud.SudoPolicy)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`#cloud-config
hostname: %s
manage_etc_hosts: true
users:
  - name: %s
    shell: /bin/bash
`, ud.Hostname, username))

	if sudoLine := cloudInitSudoLine(sudoPolicy); sudoLine != "" {
		sb.WriteString(fmt.Sprintf("    sudo: %s\n", sudoLine))
	}

	sb.WriteString(fmt.Sprintf(`    ssh-authorized-keys:
      - %s
`, ud.SSHKey))

	if len(ud.Env) > 0 {
		keys := make([]string, 0, len(ud.Env))
		for key := range ud.Env {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		sb.WriteString(`write_files:
  - path: /etc/profile.d/yeast-env.sh
    permissions: "0644"
    content: |
`)
		for _, key := range keys {
			sb.WriteString("      export " + key + "=" + shellQuote(ud.Env[key]) + "\n")
		}
	}

	return sb.String()
}

func normalizeSudoPolicy(policy SudoPolicy) SudoPolicy {
	switch strings.ToLower(strings.TrimSpace(string(policy))) {
	case string(SudoPassword):
		return SudoPassword
	case string(SudoNoPass):
		return SudoNoPass
	default:
		return SudoNone
	}
}

func cloudInitSudoLine(policy SudoPolicy) string {
	switch policy {
	case SudoPassword:
		return "ALL=(ALL) ALL"
	case SudoNoPass:
		return "ALL=(ALL) NOPASSWD:ALL"
	default:
		return ""
	}
}

func GenerateMetaData(instanceID, localHostname string) string {
	return fmt.Sprintf(`instance-id: %s
local-hostname: %s
`, instanceID, localHostname)
}

func CreateISO(dir string, userData, metaData string) error {
	// Write files
	if err := os.WriteFile(filepath.Join(dir, "user-data"), []byte(userData), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "meta-data"), []byte(metaData), 0644); err != nil {
		return err
	}

	// Create ISO
	// Check if genisoimage exists
	if _, err := exec.LookPath("genisoimage"); err != nil {
		return fmt.Errorf("genisoimage not found: please install cdrkit or genisoimage")
	}

	cmd := exec.Command("genisoimage",
		"-output", "seed.iso",
		"-volid", "cidata",
		"-joliet", "-rock",
		"user-data", "meta-data",
	)
	cmd.Dir = dir

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create ISO: %s: %s", err, string(output))
	}
	return nil
}

func LoadSSHKey() (string, error) {
	home, _ := os.UserHomeDir()
	keyPath := filepath.Join(home, ".ssh", "id_rsa.pub")
	// Try ed25519 if rsa fails
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		keyPath = filepath.Join(home, ".ssh", "id_ed25519.pub")
	}

	content, err := os.ReadFile(keyPath)
	if err != nil {
		return "", fmt.Errorf("ssh key not found: %w", err)
	}
	return strings.TrimSpace(string(content)), nil
}

func shellQuote(value string) string {
	return strconv.Quote(value)
}
