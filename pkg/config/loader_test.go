package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAppliesDefaults(t *testing.T) {
	cfgPath := writeConfig(t, `
version: 1
instances:
  - name: web
    image: ubuntu-22.04
`)

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if len(cfg.Instances) != 1 {
		t.Fatalf("expected 1 instance, got %d", len(cfg.Instances))
	}
	if cfg.Instances[0].Memory != 512 {
		t.Fatalf("expected default memory=512, got %d", cfg.Instances[0].Memory)
	}
	if cfg.Instances[0].CPUs != 1 {
		t.Fatalf("expected default cpus=1, got %d", cfg.Instances[0].CPUs)
	}
	if cfg.Instances[0].User != "yeast" {
		t.Fatalf("expected default user=yeast, got %q", cfg.Instances[0].User)
	}
	if cfg.Instances[0].Sudo != "none" {
		t.Fatalf("expected default sudo=none, got %q", cfg.Instances[0].Sudo)
	}
}

func TestLoadRejectsMissingImage(t *testing.T) {
	cfgPath := writeConfig(t, `
version: 1
instances:
  - name: web
`)

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error for missing image")
	}
	if !strings.Contains(err.Error(), "must define an image") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadRejectsInvalidInstanceName(t *testing.T) {
	cfgPath := writeConfig(t, `
version: 1
instances:
  - name: ../escape
    image: ubuntu-22.04
`)

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error for invalid instance name")
	}
	if !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadRejectsInvalidEnvKey(t *testing.T) {
	cfgPath := writeConfig(t, `
version: 1
instances:
  - name: web
    image: ubuntu-22.04
    env:
      bad-key: test
`)

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error for invalid env key")
	}
	if !strings.Contains(err.Error(), "invalid env key") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadRejectsInvalidSudoPolicy(t *testing.T) {
	cfgPath := writeConfig(t, `
version: 1
instances:
  - name: web
    image: ubuntu-22.04
    sudo: always
`)

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error for invalid sudo policy")
	}
	if !strings.Contains(err.Error(), "invalid sudo policy") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadAcceptsParameterizedUserSecurity(t *testing.T) {
	cfgPath := writeConfig(t, `
version: 1
instances:
  - name: web
    image: ubuntu-22.04
    user: operator
    sudo: password
`)

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Instances[0].User != "operator" {
		t.Fatalf("expected user=operator, got %q", cfg.Instances[0].User)
	}
	if cfg.Instances[0].Sudo != "password" {
		t.Fatalf("expected sudo=password, got %q", cfg.Instances[0].Sudo)
	}
}

func TestLoadAcceptsAndNormalizesDiskSize(t *testing.T) {
	cfgPath := writeConfig(t, `
version: 1
instances:
  - name: web
    image: ubuntu-22.04
    disk_size: 25gb
`)

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Instances[0].DiskSize != "25G" {
		t.Fatalf("expected disk_size=25G, got %q", cfg.Instances[0].DiskSize)
	}
}

func TestLoadRejectsInvalidConfigsTableDriven(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{
			name: "unsupported version",
			content: `
version: 2
instances:
  - name: web
    image: ubuntu-22.04
`,
			wantErr: "unsupported version",
		},
		{
			name: "no instances",
			content: `
version: 1
instances: []
`,
			wantErr: "at least one instance is required",
		},
		{
			name: "duplicate instance names",
			content: `
version: 1
instances:
  - name: web
    image: ubuntu-22.04
  - name: web
    image: ubuntu-24.04
`,
			wantErr: "duplicate instance name",
		},
		{
			name: "invalid linux user",
			content: `
version: 1
instances:
  - name: web
    image: ubuntu-22.04
    user: Admin
`,
			wantErr: "invalid user",
		},
		{
			name: "env value with newline",
			content: `
version: 1
instances:
  - name: web
    image: ubuntu-22.04
    env:
      API_TOKEN: "line1\nline2"
`,
			wantErr: "contains newline",
		},
		{
			name: "invalid cpu count",
			content: `
version: 1
instances:
  - name: web
    image: ubuntu-22.04
    cpus: -1
`,
			wantErr: "invalid cpu count",
		},
		{
			name: "invalid disk size",
			content: `
version: 1
instances:
  - name: web
    image: ubuntu-22.04
    disk_size: 20GiB
`,
			wantErr: "invalid disk_size",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfgPath := writeConfig(t, tc.content)
			_, err := Load(cfgPath)
			if err == nil {
				t.Fatal("expected config validation error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("unexpected error %q, want substring %q", err, tc.wantErr)
			}
		})
	}
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "yeast.yaml")
	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)+"\n"), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}
	return path
}
