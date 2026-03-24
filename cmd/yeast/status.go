package main

import (
	"fmt"
	"os"
	"text/tabwriter"
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

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "NAME\tSTATUS\tPID\tIP\tSSH PORT")
		for _, inst := range instances {
			fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%d\n", inst.Name, inst.Status, inst.PID, inst.IP, inst.SSHPort)
		}
		w.Flush()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
