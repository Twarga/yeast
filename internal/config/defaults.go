package config

import (
	"fmt"
	"strings"
)

const (
	DefaultMemoryMB       = 512
	DefaultCPUs           = 1
	DefaultUser           = "yeast"
	DefaultSudo           = "nopasswd"
	DefaultManagementHost = "127.0.0.1"
	DefaultDisplayType    = "vnc"
	DefaultDisplayHost    = "127.0.0.1"
	DefaultDisplayGPU     = "virtio-vga"
	DefaultMountDriver    = "auto"
)

func ApplyDefaults(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is required")
	}

	if strings.TrimSpace(cfg.ManagementHost) == "" {
		cfg.ManagementHost = DefaultManagementHost
	}

	for i := range cfg.Instances {
		instance := &cfg.Instances[i]

		if instance.Memory == 0 {
			instance.Memory = DefaultMemoryMB
		}
		if instance.CPUs == 0 {
			instance.CPUs = DefaultCPUs
		}
		if strings.TrimSpace(instance.Hostname) == "" {
			instance.Hostname = instance.Name
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
		for j := range instance.Ports {
			if strings.TrimSpace(instance.Ports[j].Host) == "" {
				instance.Ports[j].Host = DefaultPortForwardHost
			}
			if strings.TrimSpace(instance.Ports[j].Protocol) == "" {
				instance.Ports[j].Protocol = DefaultPortForwardProtocol
			} else {
				instance.Ports[j].Protocol = strings.ToLower(strings.TrimSpace(instance.Ports[j].Protocol))
			}
		}

		instance.Firmware = strings.ToLower(strings.TrimSpace(instance.Firmware))
		if instance.Firmware == "" {
			instance.Firmware = "bios"
		}
		if instance.Display != nil {
			display := instance.Display
			if !display.Enabled && (strings.TrimSpace(display.Type) != "" || strings.TrimSpace(display.GPU) != "" || strings.TrimSpace(display.Resolution) != "" || display.Port > 0) {
				display.Enabled = true
			}
			display.Type = strings.ToLower(strings.TrimSpace(display.Type))
			if display.Type == "" {
				display.Type = DefaultDisplayType
			}
			display.Host = strings.TrimSpace(display.Host)
			if display.Host == "" {
				display.Host = DefaultDisplayHost
			}
			display.GPU = strings.ToLower(strings.TrimSpace(display.GPU))
			if display.GPU == "" {
				display.GPU = DefaultDisplayGPU
			}
			display.Resolution = strings.ToLower(strings.TrimSpace(display.Resolution))
		}
		for j := range instance.Mounts {
			mount := &instance.Mounts[j]
			mount.Driver = strings.ToLower(strings.TrimSpace(mount.Driver))
			if mount.Driver == "" {
				mount.Driver = DefaultMountDriver
			}
			mount.Target = strings.TrimSpace(mount.Target)
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
