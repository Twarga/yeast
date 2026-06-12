package state

type ReconcileOptions struct {
	IsProcessAlive         func(pid int) bool
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
		if instance.Status != "running" {
			continue
		}
		if instance.PID <= 0 {
			if options.FindProcessByRuntimeDir != nil && instance.RuntimeDir != "" {
				if pid, ok := options.FindProcessByRuntimeDir(name, instance.RuntimeDir); ok && pid > 0 {
					instance.PID = pid
					state.Instances[name] = instance
					changed = true
				}
			}
			continue
		}
		if isAlive(instance.PID) {
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
