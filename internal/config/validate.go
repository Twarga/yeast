package config

import (
	"fmt"
	"math"
	"net"
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
	fileModePattern     = regexp.MustCompile(`^[0-7]{3,4}$`)
)

func Validate(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is required")
	}
	if cfg.Version != SupportedVersion {
		return fmt.Errorf("unsupported version: %d (expected %d)", cfg.Version, SupportedVersion)
	}
	if strings.TrimSpace(cfg.ManagementHost) != "" {
		host := strings.TrimSpace(cfg.ManagementHost)
		if host != "127.0.0.1" && host != "0.0.0.0" {
			ip := net.ParseIP(host)
			if ip == nil || ip.To4() == nil {
				return fmt.Errorf("management_host must be a valid IPv4 address (got %q)", cfg.ManagementHost)
			}
		}
	}
	if len(cfg.Instances) == 0 {
		return fmt.Errorf("at least one instance is required")
	}
	if err := validateProvision("top-level provision", cfg.Provision); err != nil {
		return err
	}
	networkByName := make(map[string]net.IPNet, len(cfg.Networks))
	if len(cfg.Networks) > 1 {
		return fmt.Errorf("at most one project network is supported")
	}
	for _, network := range cfg.Networks {
		if strings.TrimSpace(network.Name) == "" {
			return fmt.Errorf("network name cannot be empty")
		}
		if !instanceNamePattern.MatchString(network.Name) || strings.Contains(network.Name, "..") {
			return fmt.Errorf("network name %q is invalid", network.Name)
		}
		if strings.TrimSpace(network.CIDR) == "" {
			return fmt.Errorf("network %s must define a cidr", network.Name)
		}
		_, parsedCIDR, err := net.ParseCIDR(strings.TrimSpace(network.CIDR))
		if err != nil {
			return fmt.Errorf("network %s has invalid cidr %q", network.Name, network.CIDR)
		}
		if parsedCIDR.IP.To4() == nil {
			return fmt.Errorf("network %s must use an IPv4 cidr", network.Name)
		}
		networkByName[network.Name] = *parsedCIDR
	}

	seen := make(map[string]struct{}, len(cfg.Instances))
	usedNetworkIPs := make(map[string]string)
	usedBinds := make(map[string]string)
	managementHost := resolveValidatedManagementHost(cfg)
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
		if strings.TrimSpace(instance.Hostname) != "" {
			hostname := strings.TrimSpace(instance.Hostname)
			if !instanceNamePattern.MatchString(hostname) || strings.Contains(hostname, "..") {
				return fmt.Errorf("instance %s has invalid hostname %q", instance.Name, instance.Hostname)
			}
		}

		if strings.TrimSpace(instance.Image) == "" {
			return fmt.Errorf("instance %s must define an image", instance.Name)
		}
		if instance.Memory != 0 && instance.Memory < 128 {
			return fmt.Errorf("instance %s has too little memory (min 128MB)", instance.Name)
		}
		if instance.CPUs != 0 && instance.CPUs < 1 {
			return fmt.Errorf("instance %s has invalid cpu count (min 1)", instance.Name)
		}
		if instance.SSHPort != 0 && (instance.SSHPort < 1 || instance.SSHPort > 65535) {
			return fmt.Errorf("instance %s has invalid ssh_port %d", instance.Name, instance.SSHPort)
		}
		if instance.SSHPort != 0 {
			key := portBindKey(managementHost, instance.SSHPort)
			if previous, exists := usedBinds[key]; exists {
				return fmt.Errorf("instance %s ssh_port %d conflicts with %s", instance.Name, instance.SSHPort, previous)
			}
			usedBinds[key] = fmt.Sprintf("instance %s ssh_port", instance.Name)
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
		for i, forward := range instance.Ports {
			normalized, err := validatePortForward(instance.Name, i, forward)
			if err != nil {
				return err
			}
			key := portBindKey(normalized.Host, normalized.HostPort)
			if previous, exists := usedBinds[key]; exists {
				return fmt.Errorf("instance %s ports[%d] host binding %s:%d conflicts with %s", instance.Name, i, normalized.Host, normalized.HostPort, previous)
			}
			usedBinds[key] = fmt.Sprintf("instance %s ports[%d]", instance.Name, i)
		}
		if len(instance.Networks) > 1 {
			return fmt.Errorf("instance %s can attach to at most one private network", instance.Name)
		}
		seenInstanceNetworks := make(map[string]struct{}, len(instance.Networks))
		for _, attachment := range instance.Networks {
			if strings.TrimSpace(attachment.Name) == "" {
				return fmt.Errorf("instance %s network name cannot be empty", instance.Name)
			}
			if _, exists := seenInstanceNetworks[attachment.Name]; exists {
				return fmt.Errorf("instance %s attaches network %q more than once", instance.Name, attachment.Name)
			}
			seenInstanceNetworks[attachment.Name] = struct{}{}

			networkCIDR, exists := networkByName[attachment.Name]
			if !exists {
				return fmt.Errorf("instance %s references unknown network %q", instance.Name, attachment.Name)
			}
			if strings.TrimSpace(attachment.IPv4) == "" {
				return fmt.Errorf("instance %s network %q must define ipv4", instance.Name, attachment.Name)
			}
			ip := net.ParseIP(strings.TrimSpace(attachment.IPv4))
			if ip == nil || ip.To4() == nil {
				return fmt.Errorf("instance %s network %q has invalid ipv4 %q", instance.Name, attachment.Name, attachment.IPv4)
			}
			if !networkCIDR.Contains(ip) {
				return fmt.Errorf("instance %s network %q ipv4 %q is outside cidr %s", instance.Name, attachment.Name, attachment.IPv4, networkCIDR.String())
			}
			ipv4 := ip.To4()
			if ipv4.Equal(networkCIDR.IP.To4()) || isBroadcastIPv4(networkCIDR, ipv4) {
				return fmt.Errorf("instance %s network %q ipv4 %q is reserved in cidr %s", instance.Name, attachment.Name, attachment.IPv4, networkCIDR.String())
			}
			key := attachment.Name + "|" + ipv4.String()
			if previous, exists := usedNetworkIPs[key]; exists {
				return fmt.Errorf("instance %s network %q ipv4 %q is already used by instance %s", instance.Name, attachment.Name, attachment.IPv4, previous)
			}
			usedNetworkIPs[key] = instance.Name
		}
		if err := validateProvision(fmt.Sprintf("instance %s provision", instance.Name), instance.Provision); err != nil {
			return err
		}
	}

	return nil
}

func resolveValidatedManagementHost(cfg *Config) string {
	if cfg != nil && strings.TrimSpace(cfg.ManagementHost) != "" {
		return strings.TrimSpace(cfg.ManagementHost)
	}
	return DefaultManagementHost
}

func validatePortForward(instanceName string, index int, forward PortForward) (PortForward, error) {
	normalized := forward
	if strings.TrimSpace(normalized.Host) == "" {
		normalized.Host = DefaultPortForwardHost
	}
	if strings.TrimSpace(normalized.Protocol) == "" {
		normalized.Protocol = DefaultPortForwardProtocol
	}
	normalized.Host = strings.TrimSpace(normalized.Host)
	normalized.Protocol = strings.ToLower(strings.TrimSpace(normalized.Protocol))

	if normalized.Protocol != DefaultPortForwardProtocol {
		return PortForward{}, fmt.Errorf("instance %s ports[%d] protocol %q is unsupported (tcp only)", instanceName, index, forward.Protocol)
	}
	if normalized.HostPort < 1 || normalized.HostPort > 65535 {
		return PortForward{}, fmt.Errorf("instance %s ports[%d] has invalid host_port %d", instanceName, index, normalized.HostPort)
	}
	if normalized.GuestPort < 1 || normalized.GuestPort > 65535 {
		return PortForward{}, fmt.Errorf("instance %s ports[%d] has invalid guest_port %d", instanceName, index, normalized.GuestPort)
	}
	if normalized.Host != "127.0.0.1" && normalized.Host != "0.0.0.0" {
		ip := net.ParseIP(normalized.Host)
		if ip == nil || ip.To4() == nil {
			return PortForward{}, fmt.Errorf("instance %s ports[%d] host must be a valid IPv4 address (got %q)", instanceName, index, forward.Host)
		}
	}

	return normalized, nil
}

func portBindKey(host string, port int) string {
	return host + ":" + strconv.Itoa(port)
}

func isBroadcastIPv4(network net.IPNet, ip net.IP) bool {
	networkIP := network.IP.To4()
	mask := net.IP(network.Mask).To4()
	if networkIP == nil || mask == nil || ip == nil {
		return false
	}

	broadcast := make(net.IP, net.IPv4len)
	for i := 0; i < net.IPv4len; i++ {
		broadcast[i] = networkIP[i] | ^mask[i]
	}
	return ip.Equal(broadcast)
}

func validateProvision(label string, provision *ProvisionConfig) error {
	if provision == nil {
		return nil
	}

	for i, pkg := range provision.Packages {
		if strings.TrimSpace(pkg) == "" {
			return fmt.Errorf("%s packages[%d] cannot be empty", label, i)
		}
		if strings.Contains(pkg, "\n") {
			return fmt.Errorf("%s packages[%d] contains newline", label, i)
		}
	}

	for i, file := range provision.Files {
		if strings.TrimSpace(file.Source) == "" {
			return fmt.Errorf("%s files[%d] source is required", label, i)
		}
		if strings.TrimSpace(file.Destination) == "" {
			return fmt.Errorf("%s files[%d] destination is required", label, i)
		}
		if strings.Contains(file.Source, "\n") {
			return fmt.Errorf("%s files[%d] source contains newline", label, i)
		}
		if strings.Contains(file.Destination, "\n") {
			return fmt.Errorf("%s files[%d] destination contains newline", label, i)
		}
		if strings.TrimSpace(file.Permissions) != "" && !fileModePattern.MatchString(strings.TrimSpace(file.Permissions)) {
			return fmt.Errorf("%s files[%d] has invalid permissions %q", label, i, file.Permissions)
		}
	}

	for i, command := range provision.Shell {
		if strings.TrimSpace(command) == "" {
			return fmt.Errorf("%s shell[%d] cannot be empty", label, i)
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

	var multiplier int64
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
