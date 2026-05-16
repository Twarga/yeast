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
	if instance.User != DefaultUser {
		t.Fatalf("expected default user %q, got %q", DefaultUser, instance.User)
	}
	if instance.Sudo != DefaultSudo {
		t.Fatalf("expected default sudo %q, got %q", DefaultSudo, instance.Sudo)
	}
}

func TestApplyDefaultsPreservesExplicitValues(t *testing.T) {
	cfg := &Config{
		Version: 1,
		Instances: []Instance{
			{
				Name:     "web",
				Image:    "ubuntu-24.04",
				Memory:   2048,
				CPUs:     2,
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
