package main

import (
	"errors"
	"fmt"
	"syscall"
	"time"
	"yeast/pkg/state"
)

func terminateProcess(pid int, timeout time.Duration) error {
	if pid <= 0 || !state.IsProcessRunning(pid) {
		return nil
	}

	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil && !errors.Is(err, syscall.ESRCH) {
		return fmt.Errorf("failed to send SIGTERM: %w", err)
	}

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !state.IsProcessRunning(pid) {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}

	if err := syscall.Kill(pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
		return fmt.Errorf("failed to send SIGKILL: %w", err)
	}

	killDeadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(killDeadline) {
		if !state.IsProcessRunning(pid) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("process %d still running after SIGKILL", pid)
}
