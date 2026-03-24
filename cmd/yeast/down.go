package main

import (
	"fmt"
	"yeast/pkg/state"

	"github.com/spf13/cobra"
)

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Stop all running instances",
	RunE: func(cmd *cobra.Command, args []string) error {
		lock, err := state.AcquireFileLock(stateFilePath, state.DefaultLockOptions())
		if err != nil {
			return jsonCommandError("down", "state_lock_failed", fmt.Errorf("error acquiring state lock: %w", err))
		}
		defer releaseStateLock(lock)

		s, err := state.Load(stateFilePath)
		if err != nil {
			return jsonCommandError("down", "state_load_failed", fmt.Errorf("error loading state: %w", err))
		}
		s.Reconcile()

		targets := stateInstanceNames(s)
		resultData := stopCommandData{
			Schema:  "yeast.down.v1",
			Results: make([]lifecycleResult, 0, len(targets)),
		}
		if len(targets) == 0 {
			if outputJSON {
				return jsonCommandSuccess("down", resultData)
			}
			humanWarnf("No instances to stop")
			return nil
		}
		if !outputJSON {
			humanSection("Stopping Instances")
			humanKeyValue("Count", fmt.Sprintf("%d", len(targets)))
			fmt.Println()
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
					humanErrorf("Failed to stop %s cleanly: %v", humanAccent(name), err)
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
					humanWarnf("%s is already absent from state", humanAccent(name))
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
					humanSuccessf("Stopped %s", humanAccent(name))
					humanKeyValue("PID", fmt.Sprintf("%d", before.PID))
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
				humanWarnf("%s is already stopped", humanAccent(name))
			}
		}

		s.Reconcile()
		if err := s.Save(stateFilePath); err != nil {
			return jsonCommandErrorWithData("down", "state_save_failed", fmt.Errorf("error saving state: %w", err), resultData)
		}

		if resultData.Failed > 0 {
			return jsonCommandErrorWithData("down", "instance_stop_failed", fmt.Errorf("one or more instances failed to stop"), resultData)
		}
		if outputJSON {
			return jsonCommandSuccess("down", resultData)
		}
		fmt.Println()
		humanSuccessf("Stop completed")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(downCmd)
}
