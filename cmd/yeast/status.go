package main

import (
	"fmt"
	"yeast/pkg/state"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "List running instances",
	RunE: func(cmd *cobra.Command, args []string) error {
		lock, err := state.AcquireFileLock(stateFilePath, state.DefaultLockOptions())
		if err != nil {
			return jsonCommandError("status", "state_lock_failed", fmt.Errorf("error acquiring state lock: %w", err))
		}
		defer releaseStateLock(lock)

		s, err := state.Load(stateFilePath)
		if err != nil {
			return jsonCommandError("status", "state_load_failed", fmt.Errorf("error loading state: %w", err))
		}
		s.Reconcile()

		names := stateInstanceNames(s)
		instances := make([]instanceStatus, 0, len(names))
		for _, name := range names {
			inst := s.Instances[name]
			instances = append(instances, instanceStatus{
				Name:    inst.Name,
				Status:  inst.Status,
				PID:     inst.PID,
				IP:      inst.IP,
				SSHPort: inst.SSHPort,
			})
		}

		if outputJSON {
			return jsonCommandSuccess("status", statusCommandData{
				Schema:        "yeast.status.v1",
				StatePath:     stateFilePath,
				InstanceCount: len(instances),
				Instances:     instances,
			})
		}

		if len(instances) == 0 {
			humanWarnf("No tracked instances found")
			humanKeyValue("State", stateFilePath)
			return nil
		}

		humanSection("Instance Status")
		humanKeyValue("Count", fmt.Sprintf("%d", len(instances)))
		humanKeyValue("State", stateFilePath)
		fmt.Println()
		for i, inst := range instances {
			humanInfof("%s", humanAccent(inst.Name))
			humanKeyValue("Status", humanStatusLabel(inst.Status))
			humanKeyValue("PID", fmt.Sprintf("%d", inst.PID))
			humanKeyValue("IP", inst.IP)
			humanKeyValue("SSH", fmt.Sprintf("%d", inst.SSHPort))
			if i < len(instances)-1 {
				fmt.Println()
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
