package main

import (
	"fmt"
	"yeast/pkg/config"
	"yeast/pkg/state"

	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:   "restart [instance...]",
	Short: "Restart specific instances (or all configured instances when omitted)",
	RunE: func(cmd *cobra.Command, args []string) error {
		network, err := networkOptionsFromFlags()
		if err != nil {
			return jsonCommandError("restart", "invalid_network_flags", err)
		}

		cfg, err := config.Load("yeast.yaml")
		if err != nil {
			return jsonCommandError("restart", "config_load_failed", fmt.Errorf("error loading config: %w", err))
		}

		cfgByName := make(map[string]config.Instance, len(cfg.Instances))
		ordered := make([]string, 0, len(cfg.Instances))
		for _, instance := range cfg.Instances {
			cfgByName[instance.Name] = instance
			ordered = append(ordered, instance.Name)
		}

		targets := uniqueNames(args)
		if len(targets) == 0 {
			targets = ordered
		}
		if len(targets) == 0 {
			if outputJSON {
				return jsonCommandSuccess("restart", restartCommandData{
					Schema:  "yeast.restart.v1",
					Results: []lifecycleResult{},
				})
			}
			fmt.Println("No instances to restart.")
			return nil
		}

		lock, err := state.AcquireFileLock(stateFilePath, state.DefaultLockOptions())
		if err != nil {
			return jsonCommandError("restart", "state_lock_failed", fmt.Errorf("error acquiring state lock: %w", err))
		}
		defer releaseStateLock(lock)

		s, err := state.Load(stateFilePath)
		if err != nil {
			return jsonCommandError("restart", "state_load_failed", fmt.Errorf("error loading state: %w", err))
		}
		s.Reconcile()

		resultData := restartCommandData{
			Schema:  "yeast.restart.v1",
			Results: make([]lifecycleResult, 0, len(targets)),
		}
		for _, name := range targets {
			instanceCfg, ok := cfgByName[name]
			if !ok {
				resultData.NotDefined++
				resultData.Failed++
				resultData.Results = append(resultData.Results, lifecycleResult{
					Name:    name,
					Action:  "not_defined",
					Message: "instance is not defined in yeast.yaml",
				})
				if !outputJSON {
					fmt.Printf("Instance %s is not defined in yeast.yaml\n", name)
				}
				continue
			}

			before, exists := s.Instances[name]
			outcome, err := stopInstanceInState(s, name, stopGraceTimeout)
			if err != nil {
				resultData.Failed++
				resultData.Results = append(resultData.Results, lifecycleResult{
					Name:    name,
					Action:  "failed",
					Message: fmt.Sprintf("failed to stop before restart: %v", err),
				})
				if !outputJSON {
					fmt.Printf("Failed to stop %s before restart: %v\n", name, err)
				}
				continue
			}

			if !outputJSON && outcome.Exists && exists && before.Status == "running" && outcome.WasRunning {
				fmt.Printf("Stopped %s (PID %d) for restart.\n", name, before.PID)
			}

			if !outputJSON {
				fmt.Printf("Starting %s...\n", name)
			}
			inst, err := startInstanceFromConfig(instanceCfg, network)
			if err != nil {
				resultData.Failed++
				resultData.Results = append(resultData.Results, lifecycleResult{
					Name:    name,
					Action:  "failed",
					Message: fmt.Sprintf("failed to start after stop: %v", err),
				})
				if !outputJSON {
					fmt.Printf("  Failed to restart %s: %v\n", name, err)
				}
				continue
			}

			s.Instances[name] = inst
			resultData.Restarted++
			resultData.Results = append(resultData.Results, lifecycleResult{
				Name:    name,
				Action:  "restarted",
				PID:     inst.PID,
				SSHPort: inst.SSHPort,
			})
			if !outputJSON {
				fmt.Printf("  Restarted %s (PID %d, SSH Port %d)\n", name, inst.PID, inst.SSHPort)
			}
		}

		if err := s.Save(stateFilePath); err != nil {
			return jsonCommandErrorWithData("restart", "state_save_failed", fmt.Errorf("error saving state: %w", err), resultData)
		}
		if resultData.Failed > 0 {
			return jsonCommandErrorWithData("restart", "instance_restart_failed", fmt.Errorf("%d instance(s) failed to restart", resultData.Failed), resultData)
		}
		if outputJSON {
			return jsonCommandSuccess("restart", resultData)
		}
		fmt.Println("Restart completed.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(restartCmd)
	bindNetworkFlags(restartCmd)
}
