package guest

import (
	"fmt"
	"net"
)

func SSHAddress(host string, port int) (string, error) {
	if host == "" {
		return "", fmt.Errorf("host is required")
	}
	if port <= 0 {
		return "", fmt.Errorf("port must be greater than zero")
	}
	return net.JoinHostPort(host, fmt.Sprintf("%d", port)), nil
}
