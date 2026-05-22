package cloudinit

import (
	"fmt"
	"net/netip"
	"strings"

	"gopkg.in/yaml.v3"
)

type NetworkConfigInput struct {
	InterfaceName string
	MACAddress    string
	IPv4          netip.Addr
	CIDR          netip.Prefix
}

type networkConfig struct {
	Version   int                         `yaml:"version"`
	Ethernets map[string]ethernetConfigV2 `yaml:"ethernets"`
}

type ethernetConfigV2 struct {
	Match     ethernetMatch `yaml:"match"`
	SetName   string        `yaml:"set-name"`
	DHCP4     bool          `yaml:"dhcp4"`
	Addresses []string      `yaml:"addresses"`
}

type ethernetMatch struct {
	MACAddress string `yaml:"macaddress"`
}

func RenderNetworkConfig(input NetworkConfigInput) (string, error) {
	if strings.TrimSpace(input.InterfaceName) == "" {
		return "", fmt.Errorf("interface name is required")
	}
	if strings.TrimSpace(input.MACAddress) == "" {
		return "", fmt.Errorf("mac address is required")
	}
	if !input.IPv4.IsValid() || !input.IPv4.Is4() {
		return "", fmt.Errorf("valid ipv4 address is required")
	}
	if !input.CIDR.IsValid() {
		return "", fmt.Errorf("valid cidr is required")
	}
	if !input.CIDR.Contains(input.IPv4) {
		return "", fmt.Errorf("ipv4 %s is outside cidr %s", input.IPv4, input.CIDR)
	}

	body, err := yaml.Marshal(networkConfig{
		Version: 2,
		Ethernets: map[string]ethernetConfigV2{
			input.InterfaceName: {
				Match: ethernetMatch{
					MACAddress: strings.ToLower(strings.TrimSpace(input.MACAddress)),
				},
				SetName:   input.InterfaceName,
				DHCP4:     false,
				Addresses: []string{fmt.Sprintf("%s/%d", input.IPv4, input.CIDR.Bits())},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("marshal cloud-init network-config: %w", err)
	}
	return string(body), nil
}
