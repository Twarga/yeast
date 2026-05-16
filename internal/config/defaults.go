package config

import (
	"fmt"
	"strings"
)

const (
	DefaultMemoryMB = 512
	DefaultCPUs     = 1
	DefaultUser     = "yeast"
	DefaultSudo     = "none"
)

func ApplyDefaults(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is required")
	}

	for i := range cfg.Instances {
		instance := &cfg.Instances[i]

		if instance.Memory == 0 {
			instance.Memory = DefaultMemoryMB
		}
		if instance.CPUs == 0 {
			instance.CPUs = DefaultCPUs
		}
		if strings.TrimSpace(instance.User) == "" {
			instance.User = DefaultUser
		}
		if strings.TrimSpace(instance.Sudo) == "" {
			instance.Sudo = DefaultSudo
		}
		if strings.TrimSpace(instance.DiskSize) != "" {
			normalized, err := normalizeByteSize(instance.DiskSize)
			if err != nil {
				return fmt.Errorf("instance %s normalize disk_size: %w", instance.Name, err)
			}
			instance.DiskSize = normalized
		}
	}

	return nil
}

func normalizeByteSize(raw string) (string, error) {
	matches := byteSizePattern.FindStringSubmatch(strings.TrimSpace(raw))
	if len(matches) != 3 {
		return "", fmt.Errorf("unsupported size %q", raw)
	}

	suffix := strings.ToUpper(matches[2])
	if suffix == "" {
		return matches[1], nil
	}
	return matches[1] + suffix, nil
}

