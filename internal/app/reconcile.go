package app

import (
	"context"
	rtm "yeast/internal/runtime"
	"yeast/internal/state"
)

func (s *Service) reconcileStateWithRuntime(ctx context.Context, currentState *state.State) bool {
	if currentState == nil {
		return false
	}

	return state.Reconcile(currentState, state.ReconcileOptions{
		IsProcessAlive: func(pid int) bool {
			info, err := s.runtime.Inspect(ctx, rtm.RuntimeInstance{PID: pid})
			if err != nil {
				return false
			}
			return info.State == rtm.ProcessStateRunning
		},
		FindProcessByRuntimeDir: s.findProcessByRuntimeDir(ctx),
	})
}

func (s *Service) findProcessByRuntimeDir(ctx context.Context) func(name, runtimeDir string) (int, bool) {
	finder, ok := s.runtime.(rtm.ProcessFinder)
	if !ok {
		return nil
	}

	return func(name, runtimeDir string) (int, bool) {
		targets := []rtm.CleanupTarget{{Name: name, RuntimeDir: runtimeDir}}
		results, err := finder.FindProcesses(ctx, targets)
		if err != nil || len(results) == 0 {
			return 0, false
		}
		for _, result := range results {
			if result.Name == name && result.PID > 0 {
				return result.PID, true
			}
		}
		return 0, false
	}
}
