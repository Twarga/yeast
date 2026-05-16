package config

import "testing"

func validConfig() *Config {
	return &Config{
		Version: 1,
		Instances: []Instance{
			{
				Name:     "web",
				Image:    "ubuntu-24.04",
				Memory:   1024,
				CPUs:     1,
				DiskSize: "20G",
				User:     "yeast",
				Sudo:     "none",
				Env: map[string]string{
					"APP_ENV": "dev",
				},
			},
		},
	}
}

func TestValidateRejectsUnsupportedVersion(t *testing.T) {
	cfg := validConfig()
	cfg.Version = 2
	if err := Validate(cfg); err == nil {
		t.Fatal("expected unsupported version error")
	}
}

func TestValidateRejectsNoInstances(t *testing.T) {
	cfg := validConfig()
	cfg.Instances = nil
	if err := Validate(cfg); err == nil {
		t.Fatal("expected no instances error")
	}
}

func TestValidateRejectsDuplicateNames(t *testing.T) {
	cfg := validConfig()
	cfg.Instances = append(cfg.Instances, cfg.Instances[0])
	if err := Validate(cfg); err == nil {
		t.Fatal("expected duplicate instance name error")
	}
}

func TestValidateRejectsInvalidInstanceName(t *testing.T) {
	cfg := validConfig()
	cfg.Instances[0].Name = "../escape"
	if err := Validate(cfg); err == nil {
		t.Fatal("expected invalid instance name error")
	}
}

func TestValidateRejectsMissingImage(t *testing.T) {
	cfg := validConfig()
	cfg.Instances[0].Image = ""
	if err := Validate(cfg); err == nil {
		t.Fatal("expected missing image error")
	}
}

func TestValidateRejectsLowMemory(t *testing.T) {
	cfg := validConfig()
	cfg.Instances[0].Memory = 64
	if err := Validate(cfg); err == nil {
		t.Fatal("expected low memory error")
	}
}

func TestValidateRejectsInvalidCPUCount(t *testing.T) {
	cfg := validConfig()
	cfg.Instances[0].CPUs = -1
	if err := Validate(cfg); err == nil {
		t.Fatal("expected invalid cpu count error")
	}
}

func TestValidateRejectsInvalidDiskSize(t *testing.T) {
	cfg := validConfig()
	cfg.Instances[0].DiskSize = "20GiB"
	if err := Validate(cfg); err == nil {
		t.Fatal("expected invalid disk size error")
	}
}

func TestValidateRejectsInvalidUser(t *testing.T) {
	cfg := validConfig()
	cfg.Instances[0].User = "Admin"
	if err := Validate(cfg); err == nil {
		t.Fatal("expected invalid user error")
	}
}

func TestValidateRejectsInvalidSudo(t *testing.T) {
	cfg := validConfig()
	cfg.Instances[0].Sudo = "always"
	if err := Validate(cfg); err == nil {
		t.Fatal("expected invalid sudo error")
	}
}

func TestValidateRejectsInvalidEnvKey(t *testing.T) {
	cfg := validConfig()
	cfg.Instances[0].Env = map[string]string{"bad-key": "dev"}
	if err := Validate(cfg); err == nil {
		t.Fatal("expected invalid env key error")
	}
}

func TestValidateRejectsEnvValueWithNewline(t *testing.T) {
	cfg := validConfig()
	cfg.Instances[0].Env = map[string]string{"APP_ENV": "line1\nline2"}
	if err := Validate(cfg); err == nil {
		t.Fatal("expected env newline error")
	}
}

func TestValidateAcceptsValidConfig(t *testing.T) {
	if err := Validate(validConfig()); err != nil {
		t.Fatalf("expected valid config, got %v", err)
	}
}
