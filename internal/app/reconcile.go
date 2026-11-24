package app

import (
	"context"
	"path/filepath"

	"github.com/Twarga/yeast/internal/project"
	rtm "github.com/Twarga/yeast/internal/runtime"
	"github.com/Twarga/yeast/internal/state"
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

// ReconcileProject loads the project's state and reconciles it against live
// runtime processes (dead PIDs cleared, orphaned QEMU re-discovered by
// runtime dir). Used by yeastd on startup to reattach to VMs that outlived
// a previous daemon run.
func (s *Service) ReconcileProject(ctx context.Context, projectRoot string) error {
	root := projectRoot
	if root == "" {
		root = "."
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return WrapError(ErrorCodeInternal, err.Error(), err)
	}
	metadata, err := project.LoadMetadata(abs)
	if err != nil {
		return WrapError(ErrorCodeInternal, err.Error(), err)
	}
	yeastHome, err := s.resolveYeastHome()
	if err != nil {
		return WrapError(ErrorCodeInternal, err.Error(), err)
	}
	paths, err := project.NewPaths(yeastHome, metadata)
	if err != nil {
		return WrapError(ErrorCodeInternal, err.Error(), err)
	}
	lock, err := state.Acquire(paths.StateLock, state.DefaultLockOptions())
	if err != nil {
		return WrapError(ErrorCodeInternal, err.Error(), err)
	}
	defer func() { _ = lock.Release() }()
	currentState, err := state.Load(paths.StateFile, metadata.ID)
	if err != nil {
		return WrapError(ErrorCodeInternal, err.Error(), err)
	}
	if s.reconcileStateWithRuntime(ctx, &currentState) {
		if err := state.Save(paths.StateFile, currentState); err != nil {
			return WrapError(ErrorCodeInternal, err.Error(), err)
		}
	}
	return nil
}
