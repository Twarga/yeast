package cloudinit

import (
	"fmt"
	"net/netip"
	"strings"

	"gopkg.in/yaml.v3"
)

type NetworkConfigInput struct {
	ManagementInterfaceName string
	ManagementMACAddress    string
	LabInterfaceName        string
	LabMACAddress           string
	LabIPv4                 netip.Addr
	LabCIDR                 netip.Prefix
}

type networkConfig struct {
	Version   int                         `yaml:"version"`
	Ethernets map[string]ethernetConfigV2 `yaml:"ethernets"`
}

type ethernetConfigV2 struct {
	Match     ethernetMatch `yaml:"match"`
	SetName   string        `yaml:"set-name,omitempty"`
	DHCP4     bool          `yaml:"dhcp4"`
	Addresses []string      `yaml:"addresses,omitempty"`
}

type ethernetMatch struct {
	MACAddress string `yaml:"macaddress"`
}

func RenderNetworkConfig(input NetworkConfigInput) (string, error) {
	if strings.TrimSpace(input.ManagementInterfaceName) == "" {
		return "", fmt.Errorf("management interface name is required")
	}
	if strings.TrimSpace(input.ManagementMACAddress) == "" {
		return "", fmt.Errorf("management mac address is required")
	}

	ethernets := map[string]ethernetConfigV2{
		input.ManagementInterfaceName: {
			Match: ethernetMatch{
				MACAddress: strings.ToLower(strings.TrimSpace(input.ManagementMACAddress)),
			},
			DHCP4: true,
		},
	}

	labConfigured := strings.TrimSpace(input.LabInterfaceName) != "" ||
		strings.TrimSpace(input.LabMACAddress) != "" ||
		input.LabIPv4.IsValid() ||
		input.LabCIDR.IsValid()
	if labConfigured {
		if strings.TrimSpace(input.LabInterfaceName) == "" {
			return "", fmt.Errorf("lab interface name is required")
		}
		if strings.TrimSpace(input.LabMACAddress) == "" {
			return "", fmt.Errorf("lab mac address is required")
		}
		if !input.LabIPv4.IsValid() || !input.LabIPv4.Is4() {
			return "", fmt.Errorf("valid lab ipv4 address is required")
		}
		if !input.LabCIDR.IsValid() {
			return "", fmt.Errorf("valid lab cidr is required")
		}
		if !input.LabCIDR.Contains(input.LabIPv4) {
			return "", fmt.Errorf("lab ipv4 %s is outside cidr %s", input.LabIPv4, input.LabCIDR)
		}
		ethernets[input.LabInterfaceName] = ethernetConfigV2{
			Match: ethernetMatch{
				MACAddress: strings.ToLower(strings.TrimSpace(input.LabMACAddress)),
			},
			SetName:   input.LabInterfaceName,
			DHCP4:     false,
			Addresses: []string{fmt.Sprintf("%s/%d", input.LabIPv4, input.LabCIDR.Bits())},
		}
	}

	body, err := yaml.Marshal(networkConfig{
		Version:   2,
		Ethernets: ethernets,
	})
	if err != nil {
		return "", fmt.Errorf("marshal cloud-init network-config: %w", err)
	}
	return string(body), nil
}
