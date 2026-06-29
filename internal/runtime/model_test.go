package runtime

import (
	"net/netip"
	"testing"
)

func TestNetworkPlanExpressesManagementAndLabNetworks(t *testing.T) {
	t.Parallel()

	cidr := netip.MustParsePrefix("10.10.10.0/24")
	ipv4 := netip.MustParseAddr("10.10.10.10")

	plan := MachinePlan{
		Name: "web",
		Networks: NetworkPlan{
			Management: ManagementNetworkPlan{
				SSHHost:       "127.0.0.1",
				SSHPort:       2222,
				InterfaceName: "yeastmgmt0",
				MACAddress:    "52:54:00:11:22:33",
				PortForwards: []PortForwardPlan{
					{Name: "web", Host: "127.0.0.1", HostPort: 8080, GuestPort: 80, Protocol: "tcp"},
				},
			},
			Lab: &LabNetworkPlan{
				Name:          "lab",
				CIDR:          cidr,
				IPv4:          ipv4,
				InterfaceName: "yeastlab0",
				MACAddress:    "52:54:00:aa:bb:cc",
			},
		},
	}

	if plan.Networks.Management.SSHHost != "127.0.0.1" {
		t.Fatalf("unexpected management ssh host: %q", plan.Networks.Management.SSHHost)
	}
	if plan.Networks.Management.SSHPort != 2222 {
		t.Fatalf("unexpected management ssh port: %d", plan.Networks.Management.SSHPort)
	}
	if plan.Networks.Management.InterfaceName != "yeastmgmt0" {
		t.Fatalf("unexpected management interface name: %q", plan.Networks.Management.InterfaceName)
	}
	if plan.Networks.Management.MACAddress != "52:54:00:11:22:33" {
		t.Fatalf("unexpected management mac address: %q", plan.Networks.Management.MACAddress)
	}
	if len(plan.Networks.Management.PortForwards) != 1 {
		t.Fatalf("expected 1 management port forward, got %#v", plan.Networks.Management.PortForwards)
	}
	if plan.Networks.Management.PortForwards[0].HostPort != 8080 || plan.Networks.Management.PortForwards[0].GuestPort != 80 {
		t.Fatalf("unexpected management port forward: %#v", plan.Networks.Management.PortForwards[0])
	}
	if plan.Networks.Lab == nil {
		t.Fatal("expected lab network plan")
	}
	if plan.Networks.Lab.Name != "lab" {
		t.Fatalf("unexpected lab network name: %q", plan.Networks.Lab.Name)
	}
	if plan.Networks.Lab.CIDR != cidr {
		t.Fatalf("unexpected lab network cidr: %s", plan.Networks.Lab.CIDR)
	}
	if plan.Networks.Lab.IPv4 != ipv4 {
		t.Fatalf("unexpected lab network ipv4: %s", plan.Networks.Lab.IPv4)
	}
	if plan.Networks.Lab.InterfaceName != "yeastlab0" {
		t.Fatalf("unexpected lab network interface name: %q", plan.Networks.Lab.InterfaceName)
	}
	if plan.Networks.Lab.MACAddress != "52:54:00:aa:bb:cc" {
		t.Fatalf("unexpected lab network mac address: %q", plan.Networks.Lab.MACAddress)
	}
}
