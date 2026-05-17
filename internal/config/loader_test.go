package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "yeast.yaml")

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected missing file error")
	}
	if !strings.Contains(err.Error(), "read config file") {
		t.Fatalf("expected read config file error, got %v", err)
	}
	if !strings.Contains(err.Error(), path) {
		t.Fatalf("expected missing file path in error, got %v", err)
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "yeast.yaml")
	if err := os.WriteFile(path, []byte("version: [\n"), 0644); err != nil {
		t.Fatalf("failed to write invalid yaml: %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected YAML parse error")
	}
	if !strings.Contains(err.Error(), "parse config file") {
		t.Fatalf("expected parse config file error, got %v", err)
	}
	if !strings.Contains(err.Error(), path) {
		t.Fatalf("expected config path in error, got %v", err)
	}
}

func TestLoadValidYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "yeast.yaml")
	raw := `version: 1
provision:
  packages:
    - caddy
  shell:
    - systemctl enable --now caddy
instances:
  - name: web
    hostname: web-lab
    image: ubuntu-24.04
    memory: 1024
    cpus: 1
    disk_size: 20G
    ssh_port: 2205
    user: yeast
    sudo: none
    env:
      APP_ENV: dev
    provision:
      files:
        - source: ./site
          destination: /srv/site
          permissions: "0644"
`
	if err := os.WriteFile(path, []byte(raw), 0644); err != nil {
		t.Fatalf("failed to write valid yaml: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Version != 1 {
		t.Fatalf("expected version 1, got %d", cfg.Version)
	}
	if len(cfg.Instances) != 1 {
		t.Fatalf("expected 1 instance, got %d", len(cfg.Instances))
	}
	if cfg.Provision == nil {
		t.Fatal("expected top-level provision to load")
	}
	if len(cfg.Provision.Packages) != 1 || cfg.Provision.Packages[0] != "caddy" {
		t.Fatalf("expected top-level provision packages to load, got %#v", cfg.Provision.Packages)
	}

	instance := cfg.Instances[0]
	if instance.Name != "web" {
		t.Fatalf("expected instance name web, got %q", instance.Name)
	}
	if instance.Image != "ubuntu-24.04" {
		t.Fatalf("expected image ubuntu-24.04, got %q", instance.Image)
	}
	if instance.Hostname != "web-lab" {
		t.Fatalf("expected hostname web-lab, got %q", instance.Hostname)
	}
	if instance.Memory != 1024 {
		t.Fatalf("expected memory 1024, got %d", instance.Memory)
	}
	if instance.CPUs != 1 {
		t.Fatalf("expected cpus 1, got %d", instance.CPUs)
	}
	if instance.DiskSize != "20G" {
		t.Fatalf("expected disk size 20G, got %q", instance.DiskSize)
	}
	if instance.SSHPort != 2205 {
		t.Fatalf("expected ssh port 2205, got %d", instance.SSHPort)
	}
	if instance.User != "yeast" {
		t.Fatalf("expected user yeast, got %q", instance.User)
	}
	if instance.Sudo != "none" {
		t.Fatalf("expected sudo none, got %q", instance.Sudo)
	}
	if instance.Env["APP_ENV"] != "dev" {
		t.Fatalf("expected env APP_ENV=dev, got %q", instance.Env["APP_ENV"])
	}
	if instance.Provision == nil {
		t.Fatal("expected instance provision to load")
	}
	if len(instance.Provision.Files) != 1 {
		t.Fatalf("expected 1 provision file, got %d", len(instance.Provision.Files))
	}
	if instance.Provision.Files[0].Destination != "/srv/site" {
		t.Fatalf("expected provision destination /srv/site, got %q", instance.Provision.Files[0].Destination)
	}
}
