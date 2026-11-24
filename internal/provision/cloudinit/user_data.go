package cloudinit

import (
	"errors"
	"fmt"
	"github.com/Twarga/yeast/internal/config"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const cloudConfigHeader = "#cloud-config\n"

var ErrNoSSHPublicKey = errors.New("no supported ssh public key found")

type UserDataInput struct {
	Hostname      string
	Instance      config.Instance
	AuthorizedKey string
	Mounts        []MountSpec
}

// MountSpec is one resolved shared folder for the cloud-init mounts module.
type MountSpec struct {
	Target   string
	Driver   string // "virtiofs" | "9p"
	Tag      string
	ReadOnly bool
}

type cloudConfig struct {
	Hostname   string           `yaml:"hostname"`
	Users      []cloudUser      `yaml:"users"`
	WriteFiles []cloudWriteFile `yaml:"write_files,omitempty"`
	Mounts     [][]string       `yaml:"mounts,omitempty"`
	Bootcmd    []string         `yaml:"bootcmd,omitempty"`
}

type cloudUser struct {
	Name              string   `yaml:"name"`
	Sudo              string   `yaml:"sudo,omitempty"`
	Shell             string   `yaml:"shell"`
	Groups            []string `yaml:"groups,omitempty"`
	SSHAuthorizedKeys []string `yaml:"ssh_authorized_keys"`
}

type cloudWriteFile struct {
	Path        string `yaml:"path"`
	Permissions string `yaml:"permissions,omitempty"`
	Content     string `yaml:"content"`
}

func DiscoverAuthorizedKey() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home directory: %w", err)
	}

	for _, candidate := range []string{"id_ed25519.pub", "id_rsa.pub"} {
		path := filepath.Join(home, ".ssh", candidate)
		key, err := readAuthorizedKey(path)
		if err == nil {
			return key, nil
		}
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		return "", err
	}

	return "", fmt.Errorf("%w: checked ~/.ssh/id_ed25519.pub and ~/.ssh/id_rsa.pub", ErrNoSSHPublicKey)
}

func RenderUserData(input UserDataInput) (string, error) {
	hostname := strings.TrimSpace(input.Hostname)
	if hostname == "" {
		return "", fmt.Errorf("hostname is required")
	}
	user := strings.TrimSpace(input.Instance.User)
	if user == "" {
		return "", fmt.Errorf("instance user is required")
	}

	if custom := strings.TrimSpace(input.Instance.UserData); custom != "" {
		return normalizeCustomUserData(custom), nil
	}

	key := strings.TrimSpace(input.AuthorizedKey)
	if key == "" {
		return "", fmt.Errorf("authorized key is required")
	}

	cfg := cloudConfig{
		Hostname: hostname,
		Users: []cloudUser{
			{
				Name:              user,
				Sudo:              cloudSudoPolicy(input.Instance.Sudo),
				Shell:             "/bin/bash",
				Groups:            []string{"sudo"},
				SSHAuthorizedKeys: []string{key},
			},
		},
	}

	if content := buildEnvScript(input.Instance.Env); content != "" {
		cfg.WriteFiles = append(cfg.WriteFiles, cloudWriteFile{
			Path:        "/etc/profile.d/yeast-env.sh",
			Permissions: "0644",
			Content:     content,
		})
	}
	if input.Instance.Egress != nil && input.Instance.Egress.Mode == "allowlist" {
		const proxyURL = "http://10.0.2.100:3128"
		cfg.WriteFiles = append(cfg.WriteFiles,
			cloudWriteFile{
				Path:        "/etc/profile.d/yeast-egress.sh",
				Permissions: "0644",
				Content: fmt.Sprintf(
					"export HTTP_PROXY=%[1]q\nexport HTTPS_PROXY=%[1]q\nexport http_proxy=%[1]q\nexport https_proxy=%[1]q\nexport NO_PROXY=\"localhost,127.0.0.1,10.0.2.0/24\"\n",
					proxyURL,
				),
			},
			cloudWriteFile{
				Path:        "/etc/apt/apt.conf.d/99yeast-egress",
				Permissions: "0644",
				Content: fmt.Sprintf(
					"Acquire::http::Proxy %[1]q;\nAcquire::https::Proxy %[1]q;\n",
					proxyURL,
				),
			},
		)
	}

	for _, mount := range input.Mounts {
		opts := "defaults,nofail"
		fstype := mount.Driver
		if mount.Driver == "9p" {
			opts = "trans=virtio,version=9p2000.L,msize=1048576,nofail"
		}
		if mount.ReadOnly {
			opts += ",ro"
		}
		cfg.Mounts = append(cfg.Mounts, []string{mount.Tag, mount.Target, fstype, opts, "0", "0"})
		if mount.Driver == "virtiofs" {
			cfg.Bootcmd = append(cfg.Bootcmd, "modprobe virtiofs || true")
		}
	}

	body, err := yaml.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("marshal cloud-init user-data: %w", err)
	}

	return cloudConfigHeader + string(body), nil
}

func readAuthorizedKey(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read ssh public key %s: %w", path, err)
	}

	key := strings.TrimSpace(string(content))
	if key == "" {
		return "", fmt.Errorf("ssh public key %s is empty", path)
	}

	return key, nil
}

func normalizeCustomUserData(content string) string {
	trimmed := strings.TrimSpace(content)
	if !strings.HasPrefix(trimmed, "#cloud-config") {
		trimmed = cloudConfigHeader + trimmed
	}
	if !strings.HasSuffix(trimmed, "\n") {
		trimmed += "\n"
	}
	return trimmed
}

func cloudSudoPolicy(policy string) string {
	switch strings.TrimSpace(policy) {
	case "nopasswd":
		return "ALL=(ALL) NOPASSWD:ALL"
	case "password":
		return "ALL=(ALL) ALL"
	default:
		return ""
	}
}

func buildEnvScript(values map[string]string) string {
	if len(values) == 0 {
		return ""
	}

	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var lines []string
	lines = append(lines, "#!/bin/sh")
	for _, key := range keys {
		lines = append(lines, fmt.Sprintf("export %s=%s", key, shellQuote(values[key])))
	}
	return strings.Join(lines, "\n") + "\n"
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}
