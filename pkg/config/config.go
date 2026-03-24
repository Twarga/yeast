package config

type Config struct {
	Version   int        `yaml:"version"`
	Instances []Instance `yaml:"instances"`
}

type Instance struct {
	Name     string            `yaml:"name"`
	Image    string            `yaml:"image"`
	Memory   int               `yaml:"memory"`
	CPUs     int               `yaml:"cpus"`
	User     string            `yaml:"user"`
	Sudo     string            `yaml:"sudo"` // none, password, nopasswd
	UserData string            `yaml:"user_data"`
	Env      map[string]string `yaml:"env"`
}
