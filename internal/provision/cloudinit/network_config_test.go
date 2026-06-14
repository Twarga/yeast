package cloudinit

import (
	"net/netip"
	"strings"
	"testing"
)

func TestRenderNetworkConfigContainsStaticIPv4Definition(t *testing.T) {
	t.Parallel()

	got, err := RenderNetworkConfig(NetworkConfigInput{
		ManagementInterfaceName: "yeastmgmt0",
		ManagementMACAddress:    "52:54:00:11:22:33",
		LabInterfaceName:        "yeastlab0",
		LabMACAddress:           "52:54:00:aa:bb:cc",
		LabIPv4:                 netip.MustParseAddr("10.10.10.20"),
		LabCIDR:                 netip.MustParsePrefix("10.10.10.0/24"),
	})
	if err != nil {
		t.Fatalf("RenderNetworkConfig returned error: %v", err)
	}

	wantContains := []string{
		"version: 2",
		"yeastmgmt0:",
		`macaddress: "52:54:00:11:22:33"`,
		"dhcp4: true",
		"yeastlab0:",
		"macaddress: 52:54:00:aa:bb:cc",
		"set-name: yeastlab0",
		"dhcp4: false",
		"- 10.10.10.20/24",
	}
	for _, want := range wantContains {
		if !strings.Contains(got, want) {
			t.Fatalf("expected network-config to contain %q, got:\n%s", want, got)
		}
	}
	if strings.Contains(got, "set-name: yeastmgmt0") {
		t.Fatalf("management interface should not be renamed, got:\n%s", got)
	}
}

func TestRenderNetworkConfigRejectsIPv4OutsideCIDR(t *testing.T) {
	t.Parallel()

	_, err := RenderNetworkConfig(NetworkConfigInput{
		ManagementInterfaceName: "yeastmgmt0",
		ManagementMACAddress:    "52:54:00:11:22:33",
		LabInterfaceName:        "yeastlab0",
		LabMACAddress:           "52:54:00:aa:bb:cc",
		LabIPv4:                 netip.MustParseAddr("10.20.20.20"),
		LabCIDR:                 netip.MustParsePrefix("10.10.10.0/24"),
	})
	if err == nil {
		t.Fatal("expected outside-cidr error")
	}
}
