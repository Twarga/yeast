package config

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

const SupportedVersion = 1

var (
	instanceNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
	envKeyPattern       = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	linuxUserPattern    = regexp.MustCompile(`^[a-z_][a-z0-9_-]{0,31}$`)
	byteSizePattern     = regexp.MustCompile(`(?i)^\s*([0-9]+)\s*([kmgtp]?)(?:b)?\s*$`)
)

func Validate(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is required")
	}
	if cfg.Version != SupportedVersion {
		return fmt.Errorf("unsupported version: %d (expected %d)", cfg.Version, SupportedVersion)
	}
	if len(cfg.Instances) == 0 {
		return fmt.Errorf("at least one instance is required")
	}

	seen := make(map[string]struct{}, len(cfg.Instances))
	for _, instance := range cfg.Instances {
		if instance.Name == "" {
			return fmt.Errorf("instance name cannot be empty")
		}
		if !instanceNamePattern.MatchString(instance.Name) || strings.Contains(instance.Name, "..") {
			return fmt.Errorf("instance name %q is invalid", instance.Name)
		}
		if _, exists := seen[instance.Name]; exists {
			return fmt.Errorf("duplicate instance name: %s", instance.Name)
		}
		seen[instance.Name] = struct{}{}

		if strings.TrimSpace(instance.Image) == "" {
			return fmt.Errorf("instance %s must define an image", instance.Name)
		}
		if instance.Memory != 0 && instance.Memory < 128 {
			return fmt.Errorf("instance %s has too little memory (min 128MB)", instance.Name)
		}
		if instance.CPUs != 0 && instance.CPUs < 1 {
			return fmt.Errorf("instance %s has invalid cpu count (min 1)", instance.Name)
		}
		if strings.TrimSpace(instance.DiskSize) != "" {
			if _, err := parseByteSize(instance.DiskSize); err != nil {
				return fmt.Errorf("instance %s has invalid disk_size: %w", instance.Name, err)
			}
		}
		if strings.TrimSpace(instance.User) != "" && !linuxUserPattern.MatchString(instance.User) {
			return fmt.Errorf("instance %s has invalid user %q", instance.Name, instance.User)
		}
		if strings.TrimSpace(instance.Sudo) != "" {
			switch strings.ToLower(strings.TrimSpace(instance.Sudo)) {
			case "none", "password", "nopasswd":
			default:
				return fmt.Errorf("instance %s has invalid sudo policy %q", instance.Name, instance.Sudo)
			}
		}
		for key, value := range instance.Env {
			if !envKeyPattern.MatchString(key) {
				return fmt.Errorf("instance %s has invalid env key %q", instance.Name, key)
			}
			if strings.Contains(value, "\n") {
				return fmt.Errorf("instance %s env %q contains newline", instance.Name, key)
			}
		}
	}

	return nil
}

func parseByteSize(raw string) (int64, error) {
	matches := byteSizePattern.FindStringSubmatch(strings.TrimSpace(raw))
	if len(matches) != 3 {
		return 0, fmt.Errorf("unsupported size %q", raw)
	}

	value, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size value %q: %w", matches[1], err)
	}

	multiplier := int64(1)
	switch strings.ToUpper(matches[2]) {
	case "":
		multiplier = 1
	case "K":
		multiplier = 1024
	case "M":
		multiplier = 1024 * 1024
	case "G":
		multiplier = 1024 * 1024 * 1024
	case "T":
		multiplier = 1024 * 1024 * 1024 * 1024
	case "P":
		multiplier = 1024 * 1024 * 1024 * 1024 * 1024
	default:
		return 0, fmt.Errorf("unsupported size suffix %q", matches[2])
	}

	if value > math.MaxInt64/multiplier {
		return 0, fmt.Errorf("size %q is too large", raw)
	}
	return value * multiplier, nil
}

