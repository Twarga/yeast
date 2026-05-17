package config

type Config struct {
	Version   int              `yaml:"version"`
	Networks  []Network        `yaml:"networks,omitempty"`
	Instances []Instance       `yaml:"instances"`
	Provision *ProvisionConfig `yaml:"provision,omitempty"`
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

	// Reserved for later milestones.
	Networks  []string         `yaml:"networks,omitempty"`
	Provision *ProvisionConfig `yaml:"provision,omitempty"`
}

type Network struct {
	Name string `yaml:"name"`
	CIDR string `yaml:"cidr,omitempty"`
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
