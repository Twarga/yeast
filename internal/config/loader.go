package config

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file %s: %w", path, err)
	}

	var cfg Config
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		err = enrichYAMLError(err)
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

func enrichYAMLError(err error) error {
	message := err.Error()
	if strings.Contains(message, "field port not found") && !strings.Contains(message, `did you mean "ports"`) {
		return fmt.Errorf("%s; did you mean \"ports\"?", message)
	}
	return err
}
