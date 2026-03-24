package state

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
)

type State struct {
	Instances map[string]Instance `json:"instances"`
	lock      sync.Mutex
}

type Instance struct {
	Name    string `json:"name"`
	PID     int    `json:"pid"`
	Status  string `json:"status"` // "running", "stopped", "error"
	IP      string `json:"ip"`
	SSHPort int    `json:"ssh_port"`
}

func Load(filename string) (*State, error) {
	s := &State{
		Instances: make(map[string]Instance),
	}

	data, err := os.ReadFile(filename)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, s); err != nil {
		return nil, err
	}
	if s.Instances == nil {
		s.Instances = make(map[string]Instance)
	}

	return s, nil
}

func (s *State) Save(filename string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	tmp := filename + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}

	return os.Rename(tmp, filename)
}

func (s *State) Reconcile() {
	for name, instance := range s.Instances {
		if instance.Status == "running" {
			if !isExpectedInstanceProcess(name, instance.PID) {
				instance.Status = "stopped"
				instance.PID = 0
				s.Instances[name] = instance
			}
		}
	}
}

// IsProcessRunning returns true when the process exists and is signalable.
func IsProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, syscall.Signal(0))
	if err == nil {
		return true
	}
	if errors.Is(err, syscall.EPERM) {
		return true
	}
	if errors.Is(err, syscall.ESRCH) {
		return false
	}
	return false
}

func isExpectedInstanceProcess(name string, pid int) bool {
	if !IsProcessRunning(pid) {
		return false
	}

	cmdline, err := readProcessCmdline(pid)
	if err != nil {
		return false
	}

	// Verify we still point to a qemu process for this instance path.
	if !strings.Contains(cmdline, "qemu-system") {
		return false
	}

	home, err := os.UserHomeDir()
	if err != nil {
		// Fall back to qemu name-only check above if homedir is unavailable.
		return true
	}

	expectedDir := filepath.Join(home, ".yeast", "instances", name)
	return strings.Contains(cmdline, expectedDir)
}

func readProcessCmdline(pid int) (string, error) {
	path := fmt.Sprintf("/proc/%d/cmdline", pid)
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	raw = bytes.ReplaceAll(raw, []byte{0}, []byte(" "))
	return strings.TrimSpace(string(raw)), nil
}
