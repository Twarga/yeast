package state

type ReconcileOptions struct {
	IsProcessAlive func(pid int) bool
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
