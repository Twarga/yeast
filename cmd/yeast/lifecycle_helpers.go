package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"time"
	"yeast/pkg/config"
	"yeast/pkg/state"
	"yeast/pkg/util"
	"yeast/pkg/vm"
)

const (
	stateFilePath    = "yeast.state"
	stopGraceTimeout = 10 * time.Second
)

var (
	newMachineFn       = vm.New
	createDiskFn       = func(m *vm.Machine) error { return m.CreateDisk() }
	prepareConfigFn    = func(m *vm.Machine) error { return m.PrepareConfig() }
	startMachineFn     = func(m *vm.Machine) (int, error) { return m.Start() }
	getFreePortFn      = util.GetFreePort
	waitForSSHFn       = util.WaitForSSH
	terminateProcessFn = terminateProcess
)

type stopOutcome struct {
	Exists     bool
	WasRunning bool
}

func stopInstanceInState(s *state.State, name string, timeout time.Duration) (stopOutcome, error) {
	inst, exists := s.Instances[name]
	if !exists {
		return stopOutcome{Exists: false}, nil
	}

	wasRunning := inst.Status == "running" && inst.PID > 0 && state.IsProcessRunning(inst.PID)
	if wasRunning {
		if err := terminateProcessFn(inst.PID, timeout); err != nil {
			return stopOutcome{Exists: true, WasRunning: true}, err
		}
	}

	inst.Status = "stopped"
	inst.PID = 0
	s.Instances[name] = inst
	return stopOutcome{Exists: true, WasRunning: wasRunning}, nil
}

func startInstanceFromConfig(instanceCfg config.Instance, network vm.NetworkOptions) (state.Instance, error) {
	m := newMachineFn(
		instanceCfg.Name,
		instanceCfg.Image,
		instanceCfg.Memory,
		instanceCfg.CPUs,
		instanceCfg.UserData,
		instanceCfg.Env,
		instanceCfg.User,
		instanceCfg.Sudo,
		network,
	)

	if err := createDiskFn(m); err != nil {
		return state.Instance{}, fmt.Errorf("failed to create disk: %w", err)
	}

	if err := prepareConfigFn(m); err != nil {
		return state.Instance{}, fmt.Errorf("failed to prepare config: %w", err)
	}

	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		port, err := getFreePortFn()
		if err != nil {
			lastErr = fmt.Errorf("failed to allocate port: %w", err)
			continue
		}
		m.SSHPort = port

		pid, err := startMachineFn(m)
		if err != nil {
			lastErr = fmt.Errorf("failed to start QEMU: %w", err)
			continue
		}

		if err := waitForSSHFn(m.User, port, 45*time.Second); err != nil {
			lastErr = fmt.Errorf("vm not reachable on SSH port %d: %w", port, err)
			_ = terminateProcessFn(pid, 5*time.Second)
			continue
		}

		return state.Instance{
			Name:    instanceCfg.Name,
			PID:     pid,
			Status:  "running",
			IP:      "127.0.0.1",
			SSHPort: port,
		}, nil
	}

	return state.Instance{}, fmt.Errorf("failed to start instance after retries: %w", lastErr)
}

func stateInstanceNames(s *state.State) []string {
	names := make([]string, 0, len(s.Instances))
	for name := range s.Instances {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func uniqueNames(names []string) []string {
	out := make([]string, 0, len(names))
	for _, name := range names {
		if !slices.Contains(out, name) {
			out = append(out, name)
		}
	}
	return out
}

func instanceDir(name string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to resolve home directory: %w", err)
	}
	return filepath.Join(home, ".yeast", "instances", name), nil
}
