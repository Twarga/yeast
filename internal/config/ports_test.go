package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestPortForwardUnmarshalShortSyntax(t *testing.T) {
	var ports []PortForward
	if err := yaml.Unmarshal([]byte("- 8080:80\n"), &ports); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if len(ports) != 1 {
		t.Fatalf("expected 1 port mapping, got %d", len(ports))
	}
	if ports[0].Host != DefaultPortForwardHost {
		t.Fatalf("expected default host %q, got %q", DefaultPortForwardHost, ports[0].Host)
	}
	if ports[0].HostPort != 8080 || ports[0].GuestPort != 80 {
		t.Fatalf("unexpected short port mapping: %#v", ports[0])
	}
	if ports[0].Protocol != DefaultPortForwardProtocol {
		t.Fatalf("expected default protocol %q, got %q", DefaultPortForwardProtocol, ports[0].Protocol)
	}
}

func TestPortForwardUnmarshalObjectSyntax(t *testing.T) {
	var ports []PortForward
	raw := []byte("- name: grafana\n  host: 0.0.0.0\n  host_port: 3000\n  guest_port: 3000\n  protocol: tcp\n")
	if err := yaml.Unmarshal(raw, &ports); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if len(ports) != 1 {
		t.Fatalf("expected 1 port mapping, got %d", len(ports))
	}
	if ports[0].Name != "grafana" || ports[0].Host != "0.0.0.0" || ports[0].HostPort != 3000 || ports[0].GuestPort != 3000 || ports[0].Protocol != "tcp" {
		t.Fatalf("unexpected object port mapping: %#v", ports[0])
	}
}

func TestPortForwardUnmarshalRejectsInvalidShortSyntax(t *testing.T) {
	var ports []PortForward
	if err := yaml.Unmarshal([]byte("- 8080\n"), &ports); err == nil {
		t.Fatal("expected invalid short syntax error")
	}
}
