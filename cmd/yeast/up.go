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
		if !outputJSON {
			humanSection("Starting Instances")
			humanKeyValue("Count", fmt.Sprintf("%d", len(cfg.Instances)))
			humanKeyValue("Network", network.Mode)
			if network.Bridge != "" {
				humanKeyValue("Bridge", network.Bridge)
			}
			fmt.Println()
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
					humanWarnf("%s is already running (PID %d)", humanAccent(instanceCfg.Name), inst.PID)
				}
				continue
			}

			if !outputJSON {
				humanInfof("Starting %s", humanAccent(instanceCfg.Name))
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
					humanErrorf("Failed to start %s: %v", humanAccent(instanceCfg.Name), err)
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
				humanSuccessf("%s started", humanAccent(instanceCfg.Name))
				humanKeyValue("PID", fmt.Sprintf("%d", inst.PID))
				humanKeyValue("SSH", fmt.Sprintf("%s:%d", inst.IP, inst.SSHPort))
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
		if resultData.Started > 0 {
			fmt.Println()
			humanSuccessf("Started %d instance(s)", resultData.Started)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(upCmd)
	bindNetworkFlags(upCmd)
}
