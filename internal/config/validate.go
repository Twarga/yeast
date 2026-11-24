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
		switch instance.Firmware {
		case "", "bios", "uefi":
		default:
			return fmt.Errorf("instance %s has invalid firmware %q (expected bios or uefi)", instance.Name, instance.Firmware)
		}
		if err := validateDisplay(instance); err != nil {
			return err
		}
		if instance.Display != nil && instance.Display.Enabled && instance.Display.Port != 0 {
			displayHost := strings.TrimSpace(instance.Display.Host)
			if displayHost == "" {
				displayHost = DefaultDisplayHost
			}
			key := portBindKey(displayHost, instance.Display.Port)
			if previous, exists := usedBinds[key]; exists {
				return fmt.Errorf("instance %s display port %s:%d conflicts with %s", instance.Name, displayHost, instance.Display.Port, previous)
			}
			usedBinds[key] = fmt.Sprintf("instance %s display", instance.Name)
		}
		if err := validateMounts(instance); err != nil {
			return err
		}
		if instance.Egress != nil {
			switch strings.TrimSpace(instance.Egress.Mode) {
			case "", "open":
				if len(instance.Egress.Allowlist) > 0 {
					return fmt.Errorf("instance %s egress allowlist requires mode: allowlist", instance.Name)
				}
			case "offline":
				if len(instance.Egress.Allowlist) > 0 {
					return fmt.Errorf("instance %s egress mode offline does not take an allowlist", instance.Name)
				}
			case "allowlist":
				if len(instance.Egress.Allowlist) == 0 {
					return fmt.Errorf("instance %s egress mode allowlist requires at least one allowed domain", instance.Name)
				}
				for _, domain := range instance.Egress.Allowlist {
					if strings.TrimSpace(domain) == "" || strings.ContainsAny(domain, " \t/:") {
						return fmt.Errorf("instance %s has invalid egress allowlist domain %q", instance.Name, domain)
					}
				}
			default:
				return fmt.Errorf("instance %s has invalid egress mode %q (expected open, offline, or allowlist)", instance.Name, instance.Egress.Mode)
			}
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

var resolutionPattern = regexp.MustCompile(`^([0-9]+)x([0-9]+)$`)

// validateDisplay checks a display block. Field values are strict even when
// disabled; the host port only joins collision tracking when enabled.
func validateDisplay(instance Instance) error {
	if instance.Display == nil {
		return nil
	}
	d := instance.Display
	switch d.Type {
	case "", "vnc":
	default:
		return fmt.Errorf("instance %s has invalid display.type %q (only vnc is supported)", instance.Name, d.Type)
	}
	switch d.GPU {
	case "", "virtio-vga", "virtio-gpu", "virgl":
	default:
		return fmt.Errorf("instance %s has invalid display.gpu %q (expected virtio-vga, virtio-gpu, or virgl)", instance.Name, d.GPU)
	}
	host := strings.TrimSpace(d.Host)
	if host != "" && host != "127.0.0.1" && host != "0.0.0.0" {
		ip := net.ParseIP(host)
		if ip == nil || ip.To4() == nil {
			return fmt.Errorf("instance %s display.host must be a valid IPv4 address (got %q)", instance.Name, d.Host)
		}
	}
	if d.Port != 0 && (d.Port < 5900 || d.Port > 65535) {
		return fmt.Errorf("instance %s display.port must be 0 (auto) or in 5900..65535 (got %d)", instance.Name, d.Port)
	}
	if res := strings.TrimSpace(d.Resolution); res != "" {
		m := resolutionPattern.FindStringSubmatch(res)
		if m == nil {
			return fmt.Errorf("instance %s display.resolution must be <width>x<height> (got %q)", instance.Name, d.Resolution)
		}
		w, _ := strconv.Atoi(m[1])
		h, _ := strconv.Atoi(m[2])
		if w < 640 || w > 7680 || h < 480 || h > 4320 {
			return fmt.Errorf("instance %s display.resolution %q is out of range (640x480..7680x4320)", instance.Name, d.Resolution)
		}
	}
	return nil
}

// validateMounts checks host-directory mount blocks. Relative sources are
// resolved against the project root at plan time (internal/app); here we
// enforce shape and safety of the guest target and driver values.
func validateMounts(instance Instance) error {
	seenTargets := make(map[string]struct{}, len(instance.Mounts))
	for i, mount := range instance.Mounts {
		source := strings.TrimSpace(mount.Source)
		if source == "" {
			return fmt.Errorf("instance %s mounts[%d] source is required", instance.Name, i)
		}
		if strings.ContainsAny(source, "\n\x00") {
			return fmt.Errorf("instance %s mounts[%d] source contains an invalid character", instance.Name, i)
		}
		target := strings.TrimSpace(mount.Target)
		if target == "" {
			return fmt.Errorf("instance %s mounts[%d] target is required", instance.Name, i)
		}
		if !strings.HasPrefix(target, "/") || strings.Contains(target, "..") || strings.ContainsAny(target, "\n\x00") {
			return fmt.Errorf("instance %s mounts[%d] target must be a clean absolute guest path (got %q)", instance.Name, i, mount.Target)
		}
		if _, exists := seenTargets[target]; exists {
			return fmt.Errorf("instance %s mounts[%d] target %q is already mounted by this instance", instance.Name, i, target)
		}
		seenTargets[target] = struct{}{}
		switch mount.Driver {
		case "", "auto", "virtiofs", "9p":
		default:
			return fmt.Errorf("instance %s mounts[%d] has invalid driver %q (expected auto, virtiofs, or 9p)", instance.Name, i, mount.Driver)
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
