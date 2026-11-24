package qemu

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	rtm "github.com/Twarga/yeast/internal/runtime"
)

// Egress proxy sidecar: a detached `yeast egress-proxy` process bound to
// loopback; QEMU's guestfwd sends guest traffic to it. Pidfile
// <RuntimeDir>/egress-proxy.pid lets later invocations reap it.

const (
	egressProxyPIDFile  = "egress-proxy.pid"
	egressProxyPortFile = "egress-proxy.port"
	// EgressNetworkLogPath is the filename `yeast logs <vm> --network` reads.
	EgressNetworkLogName = "network.log"
)

// allocateEgressProxyPort reserves a free loopback port for the proxy.
func allocateEgressProxyPort() (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port, nil
}

// prepareEgress fills plan.Egress.ProxyPort for allowlist mode.
// Called before BuildCommandArgs so the netdev guestfwd is complete.
func prepareEgress(plan *rtm.MachinePlan) error {
	if plan.Egress == nil || plan.Egress.Mode != "allowlist" {
		return nil
	}
	port, err := allocateEgressProxyPort()
	if err != nil {
		return fmt.Errorf("allocate egress proxy port: %w", err)
	}
	plan.Egress.ProxyPort = port
	return nil
}

// spawnEgressSidecar starts the detached yeast egress-proxy for allowlist
// mode. Domains/hosts are read from <RuntimeDir>/egress.allowlist (one
// domain per line, written by the app layer before Start).
func spawnEgressSidecar(plan rtm.MachinePlan) error {
	if plan.Egress == nil || plan.Egress.Mode != "allowlist" {
		return nil
	}
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve yeast binary for egress proxy: %w", err)
	}
	allowlistPath := filepath.Join(plan.RuntimeDir, "egress.allowlist")
	args := []string{
		"egress-proxy",
		"--port", strconv.Itoa(plan.Egress.ProxyPort),
		"--allowlist-file", allowlistPath,
		"--log", filepath.Join(plan.RuntimeDir, EgressNetworkLogName),
	}
	cmd := exec.Command(self, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start egress proxy: %w", err)
	}
	pidPath := filepath.Join(plan.RuntimeDir, egressProxyPIDFile)
	if err := os.WriteFile(pidPath, []byte(strconv.Itoa(cmd.Process.Pid)), 0o644); err != nil {
		_ = cmd.Process.Kill()
		return fmt.Errorf("record egress proxy pid: %w", err)
	}
	if err := os.WriteFile(filepath.Join(plan.RuntimeDir, egressProxyPortFile), []byte(strconv.Itoa(plan.Egress.ProxyPort)), 0o644); err != nil {
		return fmt.Errorf("record egress proxy port: %w", err)
	}
	if err := cmd.Process.Release(); err != nil {
		return fmt.Errorf("release egress proxy handle: %w", err)
	}
	// Wait until the proxy is accepting connections before QEMU forwards.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", plan.Egress.ProxyPort), 100*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(25 * time.Millisecond)
	}
	return fmt.Errorf("egress proxy did not start listening on port %d", plan.Egress.ProxyPort)
}

// stopEgressSidecar SIGTERMs the recorded proxy pid and removes the pidfiles.
func stopEgressSidecar(runtimeDir string) {
	pidPath := filepath.Join(runtimeDir, egressProxyPIDFile)
	data, err := os.ReadFile(pidPath)
	if err == nil {
		if pid, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil {
			if proc, err := os.FindProcess(pid); err == nil {
				_ = proc.Signal(syscall.SIGTERM)
			}
		}
	}
	_ = os.Remove(pidPath)
	_ = os.Remove(filepath.Join(runtimeDir, egressProxyPortFile))
}
