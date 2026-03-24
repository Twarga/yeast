package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	instanceNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
	envKeyPattern       = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	linuxUserPattern    = regexp.MustCompile(`^[a-z_][a-z0-9_-]{0,31}$`)
)

func Load(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if err := validate(&cfg); err != nil {
		return nil, err
	}

	applyDefaults(&cfg)
	return &cfg, nil
}

func validate(cfg *Config) error {
	if cfg.Version != 1 {
		return fmt.Errorf("unsupported version: %d (expected 1)", cfg.Version)
	}

	if len(cfg.Instances) == 0 {
		return fmt.Errorf("at least one instance is required")
	}

	seen := make(map[string]bool)
	for i := range cfg.Instances {
		vm := &cfg.Instances[i]
		if vm.Name == "" {
			return fmt.Errorf("instance name cannot be empty")
		}
		if !instanceNamePattern.MatchString(vm.Name) {
			return fmt.Errorf("instance name %q is invalid (allowed: letters, digits, ., _, -)", vm.Name)
		}
		if seen[vm.Name] {
			return fmt.Errorf("duplicate instance name: %s", vm.Name)
		}
		seen[vm.Name] = true

		if strings.TrimSpace(vm.Image) == "" {
			return fmt.Errorf("instance %s must define an image", vm.Name)
		}
		if vm.Memory < 128 && vm.Memory != 0 {
			return fmt.Errorf("instance %s has too little memory (min 128MB)", vm.Name)
		}
		if vm.CPUs < 1 && vm.CPUs != 0 {
			return fmt.Errorf("instance %s has invalid cpu count (min 1)", vm.Name)
		}
		if strings.TrimSpace(vm.User) != "" && !linuxUserPattern.MatchString(vm.User) {
			return fmt.Errorf("instance %s has invalid user %q (allowed: lowercase linux username pattern)", vm.Name, vm.User)
		}

		if strings.TrimSpace(vm.Sudo) != "" {
			vm.Sudo = strings.ToLower(strings.TrimSpace(vm.Sudo))
			switch vm.Sudo {
			case "none", "password", "nopasswd":
			default:
				return fmt.Errorf("instance %s has invalid sudo policy %q (allowed: none, password, nopasswd)", vm.Name, vm.Sudo)
			}
		}

		for key, value := range vm.Env {
			if !envKeyPattern.MatchString(key) {
				return fmt.Errorf("instance %s has invalid env key %q", vm.Name, key)
			}
			if strings.Contains(value, "\n") {
				return fmt.Errorf("instance %s env %q contains newline, which is not supported", vm.Name, key)
			}
		}
	}
	return nil
}

func applyDefaults(cfg *Config) {
	for i := range cfg.Instances {
		if cfg.Instances[i].Memory == 0 {
			cfg.Instances[i].Memory = 512
		}
		if cfg.Instances[i].CPUs == 0 {
			cfg.Instances[i].CPUs = 1
		}
		if strings.TrimSpace(cfg.Instances[i].User) == "" {
			cfg.Instances[i].User = "yeast"
		}
		if strings.TrimSpace(cfg.Instances[i].Sudo) == "" {
			cfg.Instances[i].Sudo = "none"
		}
	}
}
