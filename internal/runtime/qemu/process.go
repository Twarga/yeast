package qemu

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const processPollInterval = 100 * time.Millisecond

type processHandle interface {
	PID() int
	Release() error
}

type processStarter func(ctx context.Context, name string, args []string, stdout, stderr *os.File) (processHandle, error)

var startProcess processStarter = startCommandContext
var processRunningFn = processRunning
var signalProcessFn = signalProcess
var waitForProcessExitFn = waitForProcessExit

type cmdProcess struct {
	cmd *exec.Cmd
}

func (p *cmdProcess) PID() int {
	if p == nil || p.cmd == nil || p.cmd.Process == nil {
		return 0
	}
	return p.cmd.Process.Pid
}

func (p *cmdProcess) Release() error {
	if p == nil || p.cmd == nil || p.cmd.Process == nil {
		return nil
	}
	return p.cmd.Process.Release()
}

func startCommandContext(ctx context.Context, name string, args []string, stdout, stderr *os.File) (processHandle, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	cmd := exec.Command(name, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &cmdProcess{cmd: cmd}, nil
}

func processRunning(pid int) (bool, error) {
	if pid <= 0 {
		return false, fmt.Errorf("pid must be greater than zero")
	}
	if zombie, err := processZombie(pid); err == nil && zombie {
		return false, nil
	} else if err != nil && !os.IsNotExist(err) {
		return false, err
	}

	err := syscall.Kill(pid, 0)
	if err == nil {
		return true, nil
	}
	if isNoSuchProcess(err) {
		return false, nil
	}
	return false, err
}

func signalProcess(pid int, sig syscall.Signal) error {
	if pid <= 0 {
		return fmt.Errorf("pid must be greater than zero")
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return process.Signal(sig)
}

func waitForProcessExit(ctx context.Context, pid int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		running, err := processRunning(pid)
		if err != nil {
			return err
		}
		if !running {
			return nil
		}
		if timeout >= 0 && time.Now().After(deadline) {
			return context.DeadlineExceeded
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(processPollInterval):
		}
	}
}

func isNoSuchProcess(err error) bool {
	if err == nil {
		return false
	}
	if err == syscall.ESRCH {
		return true
	}
	return strings.Contains(err.Error(), "no such process")
}

func processZombie(pid int) (bool, error) {
	statPath := filepath.Join("/proc", fmt.Sprintf("%d", pid), "stat")
	data, err := os.ReadFile(statPath)
	if err != nil {
		return false, err
	}

	content := string(data)
	end := strings.LastIndex(content, ")")
	if end == -1 || end+2 >= len(content) {
		return false, fmt.Errorf("unexpected proc stat format for pid %d", pid)
	}

	state := content[end+2]
	return state == 'Z', nil
}
