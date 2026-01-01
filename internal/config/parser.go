package config

import (
	"fmt"
	"os"
	"yeast/internal/types"

	"gopkg.in/yaml.v3"
)

// ParseConfig reads the YAML file at path and returns the ProjectConfig
func ParseConfig(path string) (*types.ProjectConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg types.ProjectConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse yaml: %w", err)
	}

	if err := validateAndSetDefaults(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func validateAndSetDefaults(cfg *types.ProjectConfig) error {
	if cfg.ProjectName == "" {
		return fmt.Errorf("project_name is required")
	}
	if len(cfg.Machines) == 0 {
		return fmt.Errorf("at least one machine must be defined")
	}

	seenNames := make(map[string]bool)

	for i := range cfg.Machines {
		m := &cfg.Machines[i]

		if m.Name == "" {
			return fmt.Errorf("machine at index %d is missing a name", i)
		}
		if seenNames[m.Name] {
			return fmt.Errorf("duplicate machine name: %s", m.Name)
		}
		seenNames[m.Name] = true

		if m.Image == "" {
			return fmt.Errorf("machine '%s' is missing an image", m.Name)
		}

		// Defaults
		if m.Specs.CPUs == 0 {
			m.Specs.CPUs = 1
		}
		if m.Specs.Memory == "" {
			m.Specs.Memory = "1G"
		}
	}
	return nil
}
