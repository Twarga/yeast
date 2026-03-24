package main

import (
	"fmt"
	"yeast/pkg/state"

	"github.com/spf13/cobra"
)

var haltCmd = &cobra.Command{
	Use:   "halt [instance...]",
	Short: "Stop specific instances (or all tracked instances when omitted)",
	RunE: func(cmd *cobra.Command, args []string) error {
		lock, err := state.AcquireFileLock(stateFilePath, state.DefaultLockOptions())
		if err != nil {
			return jsonCommandError("halt", "state_lock_failed", fmt.Errorf("error acquiring state lock: %w", err))
		}
		defer releaseStateLock(lock)

		s, err := state.Load(stateFilePath)
		if err != nil {
			return jsonCommandError("halt", "state_load_failed", fmt.Errorf("error loading state: %w", err))
		}
		s.Reconcile()

		targets := uniqueNames(args)
		if len(targets) == 0 {
			targets = stateInstanceNames(s)
		}
		resultData := stopCommandData{
			Schema:  "yeast.halt.v1",
			Results: make([]lifecycleResult, 0, len(targets)),
		}
		if len(targets) == 0 {
			if outputJSON {
				return jsonCommandSuccess("halt", resultData)
			}
			fmt.Println("No instances to halt.")
			return nil
		}

		for _, name := range targets {
			before, exists := s.Instances[name]
			outcome, err := stopInstanceInState(s, name, stopGraceTimeout)
			if err != nil {
				resultData.Failed++
				resultData.Results = append(resultData.Results, lifecycleResult{
					Name:    name,
					Action:  "failed",
					Message: err.Error(),
				})
				if !outputJSON {
					fmt.Printf("Failed to halt %s: %v\n", name, err)
				}
				continue
			}

			if !outcome.Exists {
				resultData.Absent++
				resultData.Results = append(resultData.Results, lifecycleResult{
					Name:    name,
					Action:  "absent",
					Message: "instance is already absent from state",
				})
				if !outputJSON {
					fmt.Printf("Instance %s is already absent from state.\n", name)
				}
				continue
			}
			if exists && before.Status == "running" && outcome.WasRunning {
				resultData.Stopped++
				resultData.Results = append(resultData.Results, lifecycleResult{
					Name:   name,
					Action: "stopped",
					PID:    before.PID,
				})
				if !outputJSON {
					fmt.Printf("Halted %s (PID %d).\n", name, before.PID)
				}
				continue
			}
			resultData.AlreadyStopped++
			resultData.Results = append(resultData.Results, lifecycleResult{
				Name:    name,
				Action:  "already_stopped",
				Message: "instance is already stopped",
			})
			if !outputJSON {
				fmt.Printf("Instance %s is already stopped.\n", name)
			}
		}

		s.Reconcile()
		if err := s.Save(stateFilePath); err != nil {
			return jsonCommandErrorWithData("halt", "state_save_failed", fmt.Errorf("error saving state: %w", err), resultData)
		}
		if resultData.Failed > 0 {
			return jsonCommandErrorWithData("halt", "instance_stop_failed", fmt.Errorf("%d instance(s) failed to halt", resultData.Failed), resultData)
		}
		if outputJSON {
			return jsonCommandSuccess("halt", resultData)
		}
		fmt.Println("Halt completed.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(haltCmd)
}
