package types

// ProjectConfig represents the root configuration in yeast.yaml
type ProjectConfig struct {
	ProjectName string    `yaml:"project_name"`
	Machines    []Machine `yaml:"machines"`
}

// Machine defines a single virtual machine configuration
type Machine struct {
	Name     string `yaml:"name"`
	Image    string `yaml:"image"`
	Template string `yaml:"template"` // Path to cloud-init template file
	Specs    Specs  `yaml:"specs"`
}

// Specs defines the hardware resources for a machine
type Specs struct {
	CPUs   int    `yaml:"cpus"`   // Number of cores, defaults to 1 if 0
	Memory string `yaml:"memory"` // e.g., "512M", "2G", defaults to "1G" if empty
}
