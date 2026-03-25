package util

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"sync"
	"time"
)

var portLock sync.Mutex

// GetFreePort asks the kernel for a free port
func GetFreePort() (int, error) {
	portLock.Lock()
	defer portLock.Unlock()

	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()

	return l.Addr().(*net.TCPAddr).Port, nil
}

// WaitForSSH waits until the SSH service is reachable and accepts a real client login probe.
func WaitForSSH(user string, port int, timeout time.Duration) error {
	if user == "" {
		user = "yeast"
	}

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 500*time.Millisecond)
		if err == nil {
			conn.Close()
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	sshPath, err := exec.LookPath("ssh")
	if err != nil {
		return fmt.Errorf("ssh client not found in PATH: %w", err)
	}

	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		// #nosec G204 -- invokes the trusted local ssh client with explicit readiness-probe arguments.
		cmd := exec.CommandContext(ctx, sshPath,
			"-p", strconv.Itoa(port),
			"-o", "BatchMode=yes",
			"-o", "ConnectTimeout=2",
			"-o", "LogLevel=ERROR",
			"-o", "StrictHostKeyChecking=no",
			"-o", "UserKnownHostsFile=/dev/null",
			"-o", "PasswordAuthentication=no",
			"-o", "KbdInteractiveAuthentication=no",
			"-o", "NumberOfPasswordPrompts=0",
			fmt.Sprintf("%s@127.0.0.1", user),
			"true",
		)
		err := cmd.Run()
		cancel()
		if err == nil {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("timed out waiting for SSH login on port %d for user %s", port, user)
}
