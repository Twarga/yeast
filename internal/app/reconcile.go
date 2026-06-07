package app

import (
	"context"
	rtm "yeast/internal/runtime"
	"yeast/internal/state"
)

func (s *Service) reconcileProjectState(ctx context.Context, currentState *state.State) (bool, error) {
	if currentState == nil {
		return false, nil
	}
	changed := state.Reconcile(currentState, state.ReconcileOptions{
		IsProcessAlive: func(pid int) bool {
			info, err := s.runtime.Inspect(ctx, rtm.RuntimeInstance{PID: pid})
			if err != nil {
				return false
			}
			return info.State == rtm.ProcessStateRunning
		},
	})

	targets := make([]rtm.CleanupTarget, 0, len(currentState.Instances))
	for name, instance := range currentState.Instances {
		if instance.Status != "running" {
			continue
		}
		targets = append(targets, rtm.CleanupTarget{
			Name:       name,
			RuntimeDir: instance.RuntimeDir,
			SSHHost:    defaultManagementHost,
			SSHPort:    instance.SSHPort,
		})
	}
	found, err := s.findQEMUProcesses(ctx, targets)
	if err != nil {
		return changed, err
	}
	if len(found) == 0 && !s.runtimeSupportsProcessFinder() {
		return changed, nil
	}
	foundByName := make(map[string]int, len(found))
	for _, process := range found {
		foundByName[process.Name] = process.PID
	}
	for name, instance := range currentState.Instances {
		if instance.Status != "running" {
			continue
		}
		pid, ok := foundByName[name]
		if ok && pid > 0 {
			if instance.PID != pid {
				instance.PID = pid
				currentState.Instances[name] = instance
				changed = true
			}
			continue
		}
		if instance.PID <= 0 || instance.SSHPort > 0 || instance.RuntimeDir != "" {
			instance.Status = "stopped"
			instance.PID = 0
			instance.ManagementIP = ""
			instance.SSHPort = 0
			instance.LastError = "process not running"
			currentState.Instances[name] = instance
			changed = true
		}
	}
	return changed, nil
}

func (s *Service) runtimeSupportsProcessFinder() bool {
	_, ok := s.runtime.(processFinder)
	return ok
}
