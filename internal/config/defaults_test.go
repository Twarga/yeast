package config

import "testing"

func TestApplyDefaults(t *testing.T) {
	cfg := &Config{
		Version: 1,
		Instances: []Instance{
			{
				Name:  "web",
				Image: "ubuntu-24.04",
			},
		},
	}

	if err := Validate(cfg); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
	if err := ApplyDefaults(cfg); err != nil {
		t.Fatalf("ApplyDefaults returned error: %v", err)
	}

	instance := cfg.Instances[0]
	if instance.Memory != DefaultMemoryMB {
		t.Fatalf("expected default memory %d, got %d", DefaultMemoryMB, instance.Memory)
	}
	if instance.CPUs != DefaultCPUs {
		t.Fatalf("expected default cpus %d, got %d", DefaultCPUs, instance.CPUs)
	}
	if instance.Hostname != "web" {
		t.Fatalf("expected default hostname web, got %q", instance.Hostname)
	}
	if instance.User != DefaultUser {
		t.Fatalf("expected default user %q, got %q", DefaultUser, instance.User)
	}
	if instance.Sudo != "nopasswd" {
		t.Fatalf("expected default sudo nopasswd, got %q", instance.Sudo)
	}
}

func TestApplyDefaultsPreservesExplicitValues(t *testing.T) {
	cfg := &Config{
		Version: 1,
		Instances: []Instance{
			{
				Name:     "web",
				Hostname: "web-lab",
				Image:    "ubuntu-24.04",
				Memory:   2048,
				CPUs:     2,
				SSHPort:  2205,
				User:     "operator",
				Sudo:     "nopasswd",
				DiskSize: "25 gb",
			},
		},
	}

	if err := Validate(cfg); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
	if err := ApplyDefaults(cfg); err != nil {
		t.Fatalf("ApplyDefaults returned error: %v", err)
	}

	instance := cfg.Instances[0]
	if instance.Memory != 2048 {
		t.Fatalf("expected explicit memory 2048, got %d", instance.Memory)
	}
	if instance.CPUs != 2 {
		t.Fatalf("expected explicit cpus 2, got %d", instance.CPUs)
	}
	if instance.SSHPort != 2205 {
		t.Fatalf("expected explicit ssh port 2205, got %d", instance.SSHPort)
	}
	if instance.Hostname != "web-lab" {
		t.Fatalf("expected explicit hostname web-lab, got %q", instance.Hostname)
	}
	if instance.User != "operator" {
		t.Fatalf("expected explicit user operator, got %q", instance.User)
	}
	if instance.Sudo != "nopasswd" {
		t.Fatalf("expected explicit sudo nopasswd, got %q", instance.Sudo)
	}
	if instance.DiskSize != "25G" {
		t.Fatalf("expected normalized disk size 25G, got %q", instance.DiskSize)
	}
}

func TestApplyDefaultsManagementHost(t *testing.T) {
	cfg := &Config{
		Version: 1,
		Instances: []Instance{
			{Name: "web", Image: "ubuntu-24.04"},
		},
	}
	if err := ApplyDefaults(cfg); err != nil {
		t.Fatalf("ApplyDefaults returned error: %v", err)
	}
	if cfg.ManagementHost != DefaultManagementHost {
		t.Fatalf("expected default management_host %q, got %q", DefaultManagementHost, cfg.ManagementHost)
	}

	cfg2 := &Config{
		Version:        1,
		ManagementHost: "0.0.0.0",
		Instances: []Instance{
			{Name: "web", Image: "ubuntu-24.04"},
		},
	}
	if err := ApplyDefaults(cfg2); err != nil {
		t.Fatalf("ApplyDefaults returned error: %v", err)
	}
	if cfg2.ManagementHost != "0.0.0.0" {
		t.Fatalf("expected explicit management_host 0.0.0.0, got %q", cfg2.ManagementHost)
	}
}

func TestApplyDefaultsDisplayAndMounts(t *testing.T) {
	cfg := validConfig()
	cfg.Instances[0].Firmware = " UEFI "
	cfg.Instances[0].Display = &DisplayConfig{Enabled: true}
	cfg.Instances[0].Mounts = []MountConfig{{Source: "./d", Target: " /mnt/d "}}
	if err := ApplyDefaults(cfg); err != nil {
		t.Fatal(err)
	}
	i := cfg.Instances[0]
	if i.Firmware != "uefi" {
		t.Fatalf("firmware = %q", i.Firmware)
	}
	if i.Display.Type != DefaultDisplayType || i.Display.Host != DefaultDisplayHost || i.Display.GPU != DefaultDisplayGPU {
		t.Fatalf("display defaults = %+v", i.Display)
	}
	if i.Mounts[0].Driver != DefaultMountDriver || i.Mounts[0].Target != "/mnt/d" {
		t.Fatalf("mount defaults = %+v", i.Mounts[0])
	}
	cfg2 := validConfig()
	if err := ApplyDefaults(cfg2); err != nil {
		t.Fatal(err)
	}
	if cfg2.Instances[0].Firmware != "bios" {
		t.Fatalf("default firmware = %q", cfg2.Instances[0].Firmware)
	}
}
