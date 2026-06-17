package guest

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
)

type SSHRunFunc func(ctx context.Context, args []string) error

var RunSSH SSHRunFunc = runSSH

func SSHAddress(host string, port int) (string, error) {
	if host == "" {
		return "", fmt.Errorf("host is required")
	}
	if port <= 0 {
		return "", fmt.Errorf("port must be greater than zero")
	}
	return net.JoinHostPort(host, fmt.Sprintf("%d", port)), nil
}

func BuildSSHArgs(user, host string, port int) ([]string, error) {
	if user == "" {
		return nil, fmt.Errorf("user is required")
	}
	if host == "" {
		return nil, fmt.Errorf("host is required")
	}
	if port <= 0 {
		return nil, fmt.Errorf("port must be greater than zero")
	}

	return []string{
		"-p", fmt.Sprintf("%d", port),
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "LogLevel=ERROR",
		fmt.Sprintf("%s@%s", user, host),
	}, nil
}

func runSSH(ctx context.Context, args []string) error {
	cmd := exec.CommandContext(ctx, "ssh", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
