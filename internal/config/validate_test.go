package config

import "testing"

func validConfig() *Config {
	return &Config{
		Version: 1,
		Instances: []Instance{
			{
				Name:     "web",
				Hostname: "web",
				Image:    "ubuntu-24.04",
				Memory:   1024,
				CPUs:     1,
				DiskSize: "20G",
				SSHPort:  2222,
				User:     "yeast",
				Sudo:     "none",
				Env: map[string]string{
					"APP_ENV": "dev",
				},
			},
		},
	}
}

func validNetworkedConfig() *Config {
	cfg := validConfig()
	cfg.Networks = []Network{
		{Name: "lab", CIDR: "10.10.10.0/24"},
	}
	cfg.Instances[0].Networks = []InstanceNetwork{
		{Name: "lab", IPv4: "10.10.10.10"},
	}
	return cfg
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

func TestValidateRejectsInvalidSSHPort(t *testing.T) {
	cfg := validConfig()
	cfg.Instances[0].SSHPort = 70000
	if err := Validate(cfg); err == nil {
		t.Fatal("expected invalid ssh port error")
	}
}

func TestValidateRejectsInvalidUser(t *testing.T) {
	cfg := validConfig()
	cfg.Instances[0].User = "Admin"
	if err := Validate(cfg); err == nil {
		t.Fatal("expected invalid user error")
	}
}

func TestValidateRejectsInvalidHostname(t *testing.T) {
	cfg := validConfig()
	cfg.Instances[0].Hostname = "../bad"
	if err := Validate(cfg); err == nil {
		t.Fatal("expected invalid hostname error")
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

func TestValidateAcceptsValidProjectNetwork(t *testing.T) {
	if err := Validate(validNetworkedConfig()); err != nil {
		t.Fatalf("expected valid network config, got %v", err)
	}
}

func TestValidateAcceptsProvisionConfig(t *testing.T) {
	cfg := validConfig()
	cfg.Provision = &ProvisionConfig{
		Packages: []string{"caddy", "curl"},
		Files: []FileProvision{
			{Source: "./site", Destination: "/srv/site", Permissions: "0644"},
		},
		Shell: []string{"systemctl enable --now caddy"},
	}
	cfg.Instances[0].Provision = &ProvisionConfig{
		Packages: []string{"git"},
		Shell:    []string{"echo ready >/tmp/ready"},
	}

	if err := Validate(cfg); err != nil {
		t.Fatalf("expected valid provision config, got %v", err)
	}
}

func TestValidateRejectsEmptyProvisionPackage(t *testing.T) {
	cfg := validConfig()
	cfg.Instances[0].Provision = &ProvisionConfig{Packages: []string{"", "curl"}}

	if err := Validate(cfg); err == nil {
		t.Fatal("expected empty package error")
	}
}

func TestValidateRejectsMissingProvisionFileSource(t *testing.T) {
	cfg := validConfig()
	cfg.Instances[0].Provision = &ProvisionConfig{
		Files: []FileProvision{{Destination: "/etc/example"}},
	}

	if err := Validate(cfg); err == nil {
		t.Fatal("expected missing provision file source error")
	}
}

func TestValidateRejectsMissingProvisionFileDestination(t *testing.T) {
	cfg := validConfig()
	cfg.Provision = &ProvisionConfig{
		Files: []FileProvision{{Source: "./local-file"}},
	}

	if err := Validate(cfg); err == nil {
		t.Fatal("expected missing provision file destination error")
	}
}

func TestValidateRejectsInvalidProvisionFilePermissions(t *testing.T) {
	cfg := validConfig()
	cfg.Instances[0].Provision = &ProvisionConfig{
		Files: []FileProvision{{Source: "./local-file", Destination: "/tmp/file", Permissions: "rwx"}},
	}

	if err := Validate(cfg); err == nil {
		t.Fatal("expected invalid provision file permissions error")
	}
}

func TestValidateRejectsEmptyProvisionShellCommand(t *testing.T) {
	cfg := validConfig()
	cfg.Provision = &ProvisionConfig{
		Shell: []string{"echo ok", "   "},
	}

	if err := Validate(cfg); err == nil {
		t.Fatal("expected empty provision shell command error")
	}
}

func TestValidateRejectsTooManyProjectNetworks(t *testing.T) {
	cfg := validConfig()
	cfg.Networks = []Network{
		{Name: "lab-a", CIDR: "10.10.10.0/24"},
		{Name: "lab-b", CIDR: "10.20.20.0/24"},
	}

	if err := Validate(cfg); err == nil {
		t.Fatal("expected too many project networks error")
	}
}

func TestValidateRejectsInvalidProjectNetworkCIDR(t *testing.T) {
	cfg := validConfig()
	cfg.Networks = []Network{{Name: "lab", CIDR: "not-a-cidr"}}

	if err := Validate(cfg); err == nil {
		t.Fatal("expected invalid project network cidr error")
	}
}

func TestValidateRejectsMissingProjectNetworkCIDR(t *testing.T) {
	cfg := validConfig()
	cfg.Networks = []Network{{Name: "lab"}}

	if err := Validate(cfg); err == nil {
		t.Fatal("expected missing project network cidr error")
	}
}

func TestValidateRejectsIPv6ProjectNetworkCIDR(t *testing.T) {
	cfg := validConfig()
	cfg.Networks = []Network{{Name: "lab", CIDR: "fd00::/64"}}

	if err := Validate(cfg); err == nil {
		t.Fatal("expected ipv6 project network cidr error")
	}
}

func TestValidateRejectsUnknownInstanceNetwork(t *testing.T) {
	cfg := validConfig()
	cfg.Instances[0].Networks = []InstanceNetwork{{Name: "lab", IPv4: "10.10.10.10"}}

	if err := Validate(cfg); err == nil {
		t.Fatal("expected unknown instance network error")
	}
}

func TestValidateRejectsInvalidInstanceNetworkIPv4(t *testing.T) {
	cfg := validNetworkedConfig()
	cfg.Instances[0].Networks[0].IPv4 = "bad-ip"

	if err := Validate(cfg); err == nil {
		t.Fatal("expected invalid instance network ipv4 error")
	}
}

func TestValidateRejectsReservedInstanceNetworkIPv4(t *testing.T) {
	tests := []string{"10.10.10.0", "10.10.10.255"}

	for _, ip := range tests {
		ip := ip
		t.Run(ip, func(t *testing.T) {
			cfg := validNetworkedConfig()
			cfg.Instances[0].Networks[0].IPv4 = ip

			if err := Validate(cfg); err == nil {
				t.Fatalf("expected reserved instance network ipv4 error for %s", ip)
			}
		})
	}
}

func TestValidateRejectsInstanceNetworkIPv4OutsideCIDR(t *testing.T) {
	cfg := validNetworkedConfig()
	cfg.Instances[0].Networks[0].IPv4 = "10.99.99.10"

	if err := Validate(cfg); err == nil {
		t.Fatal("expected instance network ipv4 outside cidr error")
	}
}

func TestValidateRejectsDuplicateInstanceNetworkIPv4(t *testing.T) {
	cfg := validNetworkedConfig()
	cfg.Instances = append(cfg.Instances, Instance{
		Name:   "db",
		Image:  "ubuntu-24.04",
		Memory: 1024,
		CPUs:   1,
		User:   "yeast",
		Sudo:   "none",
		Networks: []InstanceNetwork{
			{Name: "lab", IPv4: "10.10.10.10"},
		},
	})

	if err := Validate(cfg); err == nil {
		t.Fatal("expected duplicate instance network ipv4 error")
	}
}

func TestValidateRejectsMultipleInstanceNetworkAttachments(t *testing.T) {
	cfg := validConfig()
	cfg.Networks = []Network{{Name: "lab", CIDR: "10.10.10.0/24"}}
	cfg.Instances[0].Networks = []InstanceNetwork{
		{Name: "lab", IPv4: "10.10.10.10"},
		{Name: "lab", IPv4: "10.10.10.11"},
	}

	if err := Validate(cfg); err == nil {
		t.Fatal("expected too many instance network attachments error")
	}
}
