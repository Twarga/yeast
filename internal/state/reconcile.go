package state

type ReconcileOptions struct {
	IsProcessAlive          func(pid int) bool
	FindProcessByRuntimeDir func(name, runtimeDir string) (int, bool)
}

func Reconcile(state *State, options ReconcileOptions) bool {
	if state == nil || state.Instances == nil {
		return false
	}

	isAlive := options.IsProcessAlive
	if isAlive == nil {
		isAlive = processAlive
	}

	changed := false
	for name, instance := range state.Instances {
		if instance.PID > 0 && isAlive(instance.PID) {
			if instance.Status != "running" || instance.LastError != "" {
				instance.Status = "running"
				instance.LastError = ""
				state.Instances[name] = instance
				changed = true
			}
			continue
		}

		if options.FindProcessByRuntimeDir != nil && instance.RuntimeDir != "" {
			if pid, ok := options.FindProcessByRuntimeDir(name, instance.RuntimeDir); ok && pid > 0 {
				if instance.Status != "running" || instance.PID != pid || instance.LastError != "" {
					instance.Status = "running"
					instance.PID = pid
					instance.LastError = ""
					state.Instances[name] = instance
					changed = true
				}
				continue
			}
		}

		if instance.Status != "running" {
			continue
		}

		if instance.PID <= 0 {
			continue
		}

		instance.Status = "stopped"
		instance.PID = 0
		instance.ManagementIP = ""
		instance.SSHPort = 0
		instance.LastError = "process not running"
		state.Instances[name] = instance
		changed = true
	}

	return changed
}
