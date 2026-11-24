package config

type Config struct {
	Version        int              `yaml:"version"`
	ManagementHost string           `yaml:"management_host,omitempty"`
	Networks       []Network        `yaml:"networks,omitempty"`
	Instances      []Instance       `yaml:"instances"`
	Provision      *ProvisionConfig `yaml:"provision,omitempty"`
}

type Instance struct {
	Name     string            `yaml:"name"`
	Hostname string            `yaml:"hostname,omitempty"`
	Image    string            `yaml:"image"`
	Memory   int               `yaml:"memory"`
	CPUs     int               `yaml:"cpus"`
	DiskSize string            `yaml:"disk_size,omitempty"`
	SSHPort  int               `yaml:"ssh_port,omitempty"`
	User     string            `yaml:"user,omitempty"`
	Sudo     string            `yaml:"sudo,omitempty"`
	Env      map[string]string `yaml:"env,omitempty"`
	UserData string            `yaml:"user_data,omitempty"`
	Ports    []PortForward     `yaml:"ports,omitempty"`
	Firmware string            `yaml:"firmware,omitempty"`
	Display  *DisplayConfig    `yaml:"display,omitempty"`
	Mounts   []MountConfig     `yaml:"mounts,omitempty"`

	Networks  []InstanceNetwork `yaml:"networks,omitempty"`
	Provision *ProvisionConfig  `yaml:"provision,omitempty"`
	Egress    *EgressConfig     `yaml:"egress,omitempty"`
}

// EgressConfig controls the guest's outbound network reachability.
// mode "open" (default) — unrestricted user-mode NAT;
// "offline" — restrict=on (no internet, no host); SSH/hostfwd still works;
// "allowlist" — restrict=on plus a guestfwd to Yeast's local HTTP/CONNECT
// proxy that only allows the listed domains and logs every decision.
type EgressConfig struct {
	Mode      string   `yaml:"mode,omitempty"`
	Allowlist []string `yaml:"allowlist,omitempty"`
}

// DisplayConfig describes an optional graphical display for an instance.
// Display is disabled unless enabled: true — headless stays the default.
type DisplayConfig struct {
	Enabled    bool   `yaml:"enabled"`
	Type       string `yaml:"type,omitempty"`
	Host       string `yaml:"host,omitempty"`
	Port       int    `yaml:"port,omitempty"`
	GPU        string `yaml:"gpu,omitempty"`
	Resolution string `yaml:"resolution,omitempty"`
}

// MountConfig shares a host directory into the guest.
type MountConfig struct {
	Source   string `yaml:"source"`
	Target   string `yaml:"target"`
	ReadOnly bool   `yaml:"readonly,omitempty"`
	Driver   string `yaml:"driver,omitempty"`
}

type Network struct {
	Name string `yaml:"name"`
	CIDR string `yaml:"cidr,omitempty"`
}

type InstanceNetwork struct {
	Name string `yaml:"name"`
	IPv4 string `yaml:"ipv4,omitempty"`
}

type ProvisionConfig struct {
	Packages []string        `yaml:"packages,omitempty"`
	Files    []FileProvision `yaml:"files,omitempty"`
	Shell    []string        `yaml:"shell,omitempty"`
}

type FileProvision struct {
	Source      string `yaml:"source"`
	Destination string `yaml:"destination"`
	Permissions string `yaml:"permissions,omitempty"`
}
