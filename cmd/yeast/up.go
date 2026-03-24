package main

import (
	"fmt"
	"yeast/pkg/config"
	"yeast/pkg/state"

	"github.com/spf13/cobra"
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Start all VMs defined in yeast.yaml",
	RunE: func(cmd *cobra.Command, args []string) error {
		network, err := networkOptionsFromFlags()
		if err != nil {
			return jsonCommandError("up", "invalid_network_flags", err)
		}

		cfg, err := config.Load("yeast.yaml")
		if err != nil {
			return jsonCommandError("up", "config_load_failed", fmt.Errorf("error loading config: %w", err))
		}

		lock, err := state.AcquireFileLock(stateFilePath, state.DefaultLockOptions())
		if err != nil {
			return jsonCommandError("up", "state_lock_failed", fmt.Errorf("error acquiring state lock: %w", err))
		}
		defer releaseStateLock(lock)

		s, err := state.Load(stateFilePath)
		if err != nil {
			return jsonCommandError("up", "state_load_failed", fmt.Errorf("error loading state: %w", err))
		}
		s.Reconcile()

		resultData := upCommandData{
			Schema:  "yeast.up.v1",
			Results: make([]lifecycleResult, 0, len(cfg.Instances)),
		}
		for _, instanceCfg := range cfg.Instances {
			// Check if already running
			if inst, exists := s.Instances[instanceCfg.Name]; exists && inst.Status == "running" {
				resultData.AlreadyRunning++
				resultData.Results = append(resultData.Results, lifecycleResult{
					Name:    instanceCfg.Name,
					Action:  "already_running",
					PID:     inst.PID,
					SSHPort: inst.SSHPort,
					Message: "instance is already running",
				})
				if !outputJSON {
					fmt.Printf("Instance %s is already running (PID %d)\n", instanceCfg.Name, inst.PID)
				}
				continue
			}

			if !outputJSON {
				fmt.Printf("Starting %s...\n", instanceCfg.Name)
			}
			inst, err := startInstanceFromConfig(instanceCfg, network)
			if err != nil {
				resultData.Failed++
				resultData.Results = append(resultData.Results, lifecycleResult{
					Name:    instanceCfg.Name,
					Action:  "failed",
					Message: err.Error(),
				})
				if !outputJSON {
					fmt.Printf("  Failed to start instance: %v\n", err)
				}
				continue
			}
			resultData.Started++
			resultData.Results = append(resultData.Results, lifecycleResult{
				Name:    instanceCfg.Name,
				Action:  "started",
				PID:     inst.PID,
				SSHPort: inst.SSHPort,
			})
			if !outputJSON {
				fmt.Printf("  Started! (PID %d, SSH Port %d)\n", inst.PID, inst.SSHPort)
			}
			s.Instances[instanceCfg.Name] = inst
		}

		if err := s.Save(stateFilePath); err != nil {
			return jsonCommandErrorWithData("up", "state_save_failed", fmt.Errorf("error saving state: %w", err), resultData)
		}
		if resultData.Failed > 0 {
			return jsonCommandErrorWithData("up", "instance_start_failed", fmt.Errorf("%d instance(s) failed to start", resultData.Failed), resultData)
		}
		if outputJSON {
			return jsonCommandSuccess("up", resultData)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(upCmd)
	bindNetworkFlags(upCmd)
}
