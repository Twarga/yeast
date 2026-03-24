package main

import (
	"errors"
	"fmt"
	"os"
	"yeast/pkg/state"

	"github.com/spf13/cobra"
)

var destroyCmd = &cobra.Command{
	Use:   "destroy [instance...]",
	Short: "Stop and delete local instance data",
	RunE: func(cmd *cobra.Command, args []string) error {
		lock, err := state.AcquireFileLock(stateFilePath, state.DefaultLockOptions())
		if err != nil {
			return jsonCommandError("destroy", "state_lock_failed", fmt.Errorf("error acquiring state lock: %w", err))
		}
		defer releaseStateLock(lock)

		s, err := state.Load(stateFilePath)
		if err != nil {
			return jsonCommandError("destroy", "state_load_failed", fmt.Errorf("error loading state: %w", err))
		}
		s.Reconcile()

		targets := uniqueNames(args)
		if len(targets) == 0 {
			targets = stateInstanceNames(s)
		}
		resultData := destroyCommandData{
			Schema:  "yeast.destroy.v1",
			Results: make([]lifecycleResult, 0, len(targets)),
		}
		if len(targets) == 0 {
			if outputJSON {
				return jsonCommandSuccess("destroy", resultData)
			}
			fmt.Println("No instances to destroy.")
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
					Message: fmt.Sprintf("failed to stop before destroy: %v", err),
				})
				if !outputJSON {
					fmt.Printf("Failed to stop %s before destroy: %v\n", name, err)
				}
				continue
			}
			if !outputJSON && outcome.Exists && exists && before.Status == "running" && outcome.WasRunning {
				fmt.Printf("Stopped %s (PID %d) before destroy.\n", name, before.PID)
			}

			dir, err := instanceDir(name)
			if err != nil {
				resultData.Failed++
				resultData.Results = append(resultData.Results, lifecycleResult{
					Name:    name,
					Action:  "failed",
					Message: fmt.Sprintf("failed to resolve instance directory: %v", err),
				})
				if !outputJSON {
					fmt.Printf("Failed to resolve instance directory for %s: %v\n", name, err)
				}
				continue
			}

			_, statBeforeErr := os.Stat(dir)
			dirExisted := statBeforeErr == nil

			if err := os.RemoveAll(dir); err != nil {
				resultData.Failed++
				resultData.Results = append(resultData.Results, lifecycleResult{
					Name:    name,
					Action:  "failed",
					Message: fmt.Sprintf("failed to remove instance directory: %v", err),
				})
				if !outputJSON {
					fmt.Printf("Failed to remove instance directory for %s: %v\n", name, err)
				}
				continue
			}

			delete(s.Instances, name)

			_, statErr := os.Stat(dir)
			if statErr == nil {
				resultData.Failed++
				resultData.Results = append(resultData.Results, lifecycleResult{
					Name:    name,
					Action:  "failed",
					Message: fmt.Sprintf("instance directory still exists at %s", dir),
				})
				if !outputJSON {
					fmt.Printf("Instance directory for %s still exists at %s\n", name, dir)
				}
				continue
			}
			if !errors.Is(statErr, os.ErrNotExist) {
				resultData.Failed++
				resultData.Results = append(resultData.Results, lifecycleResult{
					Name:    name,
					Action:  "failed",
					Message: fmt.Sprintf("failed to verify instance directory removal: %v", statErr),
				})
				if !outputJSON {
					fmt.Printf("Failed to verify instance directory removal for %s: %v\n", name, statErr)
				}
				continue
			}

			if !outcome.Exists && !dirExisted {
				resultData.Absent++
				resultData.Results = append(resultData.Results, lifecycleResult{
					Name:    name,
					Action:  "absent",
					Message: "instance is already absent",
				})
				if !outputJSON {
					fmt.Printf("Instance %s is already absent.\n", name)
				}
				continue
			}

			resultData.Destroyed++
			resultData.Results = append(resultData.Results, lifecycleResult{
				Name:   name,
				Action: "destroyed",
			})
			if !outputJSON {
				fmt.Printf("Destroyed %s.\n", name)
			}
		}

		if err := s.Save(stateFilePath); err != nil {
			return jsonCommandErrorWithData("destroy", "state_save_failed", fmt.Errorf("error saving state: %w", err), resultData)
		}
		if resultData.Failed > 0 {
			return jsonCommandErrorWithData("destroy", "instance_destroy_failed", fmt.Errorf("%d instance(s) failed to destroy", resultData.Failed), resultData)
		}
		if outputJSON {
			return jsonCommandSuccess("destroy", resultData)
		}
		fmt.Println("Destroy completed.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(destroyCmd)
}
