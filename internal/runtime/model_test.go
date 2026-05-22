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
				SSHHost: "127.0.0.1",
				SSHPort: 2222,
			},
			Lab: &LabNetworkPlan{
				Name: "lab",
				CIDR: cidr,
				IPv4: ipv4,
			},
		},
	}

	if plan.Networks.Management.SSHHost != "127.0.0.1" {
		t.Fatalf("unexpected management ssh host: %q", plan.Networks.Management.SSHHost)
	}
	if plan.Networks.Management.SSHPort != 2222 {
		t.Fatalf("unexpected management ssh port: %d", plan.Networks.Management.SSHPort)
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
}
