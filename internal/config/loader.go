package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parse config file %s: %w", path, err)
	}
	if err := Validate(&cfg); err != nil {
		return nil, fmt.Errorf("validate config file %s: %w", path, err)
	}
	if err := ApplyDefaults(&cfg); err != nil {
		return nil, fmt.Errorf("prepare config file %s: %w", path, err)
	}

	return &cfg, nil
}
