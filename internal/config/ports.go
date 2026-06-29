package config

import (
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	DefaultPortForwardHost     = "127.0.0.1"
	DefaultPortForwardProtocol = "tcp"
)

type PortForward struct {
	Name      string `yaml:"name,omitempty"`
	Host      string `yaml:"host,omitempty"`
	HostPort  int    `yaml:"host_port,omitempty"`
	GuestPort int    `yaml:"guest_port,omitempty"`
	Protocol  string `yaml:"protocol,omitempty"`
}

func (p *PortForward) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		raw := strings.TrimSpace(node.Value)
		hostPort, guestPort, err := parseShortPortForward(raw)
		if err != nil {
			return err
		}
		*p = PortForward{
			Host:      DefaultPortForwardHost,
			HostPort:  hostPort,
			GuestPort: guestPort,
			Protocol:  DefaultPortForwardProtocol,
		}
		return nil
	case yaml.MappingNode:
		type alias PortForward
		var decoded alias
		if err := node.Decode(&decoded); err != nil {
			return err
		}
		*p = PortForward(decoded)
		return nil
	default:
		return fmt.Errorf("port mapping must be a string like 8080:80 or an object")
	}
}

func parseShortPortForward(raw string) (int, int, error) {
	parts := strings.Split(raw, ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid port mapping %q (expected host:guest)", raw)
	}

	hostPort, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, fmt.Errorf("invalid host port in %q", raw)
	}
	guestPort, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, fmt.Errorf("invalid guest port in %q", raw)
	}

	return hostPort, guestPort, nil
}
