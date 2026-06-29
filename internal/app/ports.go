package app

import (
	"fmt"
	"strconv"
	"strings"
	"yeast/internal/config"
	rtm "yeast/internal/runtime"
	"yeast/internal/state"
)

type PortForwardResult struct {
	Name      string `json:"name,omitempty"`
	Host      string `json:"host"`
	HostPort  int    `json:"host_port"`
	GuestPort int    `json:"guest_port"`
	Protocol  string `json:"protocol"`
	URL       string `json:"url,omitempty"`
}

func buildRuntimePortForwards(forwards []config.PortForward) []rtm.PortForwardPlan {
	result := make([]rtm.PortForwardPlan, 0, len(forwards))
	for _, forward := range forwards {
		result = append(result, rtm.PortForwardPlan{
			Name:      forward.Name,
			Host:      strings.TrimSpace(forward.Host),
			HostPort:  forward.HostPort,
			GuestPort: forward.GuestPort,
			Protocol:  strings.ToLower(strings.TrimSpace(forward.Protocol)),
		})
	}
	return result
}

func buildStatePortForwards(forwards []config.PortForward) []state.PortForwardState {
	result := make([]state.PortForwardState, 0, len(forwards))
	for _, forward := range forwards {
		result = append(result, state.PortForwardState{
			Name:      forward.Name,
			Host:      strings.TrimSpace(forward.Host),
			HostPort:  forward.HostPort,
			GuestPort: forward.GuestPort,
			Protocol:  strings.ToLower(strings.TrimSpace(forward.Protocol)),
		})
	}
	return result
}

func buildPortForwardResults(forwards []state.PortForwardState) []PortForwardResult {
	result := make([]PortForwardResult, 0, len(forwards))
	for _, forward := range forwards {
		url := ""
		switch forward.GuestPort {
		case 80:
			url = fmt.Sprintf("http://%s:%d", forward.Host, forward.HostPort)
		case 443:
			url = fmt.Sprintf("https://%s:%d", forward.Host, forward.HostPort)
		default:
			url = fmt.Sprintf("%s:%d", forward.Host, forward.HostPort)
		}
		result = append(result, PortForwardResult{
			Name:      forward.Name,
			Host:      forward.Host,
			HostPort:  forward.HostPort,
			GuestPort: forward.GuestPort,
			Protocol:  forward.Protocol,
			URL:       url,
		})
	}
	return result
}

func usedHostBindings(current state.State) map[string]string {
	result := make(map[string]string)
	for name, instance := range current.Instances {
		if instance.ManagementIP != "" && instance.SSHPort > 0 {
			result[hostPortKey(instance.ManagementIP, instance.SSHPort)] = name + " ssh"
		}
		for _, forward := range instance.ServicePorts {
			result[hostPortKey(forward.Host, forward.HostPort)] = name + " port"
		}
	}
	return result
}

func reserveConfiguredPortBindings(instance config.Instance, managementHost string, sshPort int, allocated map[string]string) error {
	for _, forward := range instance.Ports {
		key := hostPortKey(forward.Host, forward.HostPort)
		if previous, exists := allocated[key]; exists {
			return fmt.Errorf("instance %s port %s:%d conflicts with %s", instance.Name, forward.Host, forward.HostPort, previous)
		}
		if forward.Host == managementHost && forward.HostPort == sshPort {
			return fmt.Errorf("instance %s port %s:%d conflicts with its ssh management port", instance.Name, forward.Host, forward.HostPort)
		}
		allocated[key] = instance.Name + " port"
	}
	return nil
}

func hostPortKey(host string, port int) string {
	return host + ":" + strconv.Itoa(port)
}
