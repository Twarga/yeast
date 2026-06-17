package qemu

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	rtm "yeast/internal/runtime"
)

func (r *Runtime) FindProcesses(ctx context.Context, targets []rtm.CleanupTarget) ([]rtm.CleanupResult, error) {
	if len(targets) == 0 {
		return nil, nil
	}

	byRuntimeDir := make(map[string]rtm.CleanupTarget, len(targets))
	for _, target := range targets {
		if strings.TrimSpace(target.RuntimeDir) == "" {
			continue
		}
		byRuntimeDir[filepath.Clean(target.RuntimeDir)] = target
	}
	if len(byRuntimeDir) == 0 {
		return nil, nil
	}

	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}

	results := make([]rtm.CleanupResult, 0)
	for _, entry := range entries {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid <= 0 {
			continue
		}

		cmdline, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "cmdline"))
		if err != nil || len(cmdline) == 0 {
			continue
		}
		commandLine := strings.ReplaceAll(string(cmdline), "\x00", " ")

		for runtimeDir, target := range byRuntimeDir {
			if !strings.Contains(commandLine, qmpSocketPath(runtimeDir)) {
				continue
			}
			running, err := processRunningFn(pid)
			if err != nil || !running {
				break
			}
			results = append(results, rtm.CleanupResult{
				Name: target.Name,
				PID:  pid,
			})
			break
		}
	}

	return results, nil
}
