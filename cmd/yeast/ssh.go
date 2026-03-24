package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"yeast/pkg/state"

	"github.com/spf13/cobra"
)

var (
	sshInsecure bool
	sshUser     string
)

var sshCmd = &cobra.Command{
	Use:   "ssh [name]",
	Short: "Connect to an instance via SSH",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if outputJSON {
			return jsonCommandError("ssh", "json_not_supported", fmt.Errorf("json output is not supported for interactive ssh sessions"))
		}

		name := args[0]
		lock, err := state.AcquireFileLock("yeast.state", state.DefaultLockOptions())
		if err != nil {
			return fmt.Errorf("error acquiring state lock: %w", err)
		}
		defer releaseStateLock(lock)

		s, err := state.Load("yeast.state")
		if err != nil {
			return fmt.Errorf("error loading state: %w", err)
		}
		s.Reconcile()

		inst, ok := s.Instances[name]
		if !ok || inst.Status != "running" {
			return fmt.Errorf("instance %s is not running", name)
		}

		sshPath, err := exec.LookPath("ssh")
		if err != nil {
			return fmt.Errorf("ssh binary not found in PATH: %w", err)
		}

		sshArgs := []string{
			"ssh",
			"-p", fmt.Sprintf("%d", inst.SSHPort),
			"-o", "LogLevel=ERROR",
		}
		if sshInsecure {
			sshArgs = append(sshArgs,
				"-o", "StrictHostKeyChecking=no",
				"-o", "UserKnownHostsFile=/dev/null",
			)
		}
		sshArgs = append(sshArgs, fmt.Sprintf("%s@%s", sshUser, inst.IP))

		fmt.Printf("Connecting to %s on port %d...\n", name, inst.SSHPort)

		// #nosec G204 -- executes trusted ssh binary with explicit args for user-requested interactive session.
		if err := syscall.Exec(sshPath, sshArgs, os.Environ()); err != nil {
			return fmt.Errorf("failed to exec ssh: %w", err)
		}
		return nil
	},
}

func init() {
	sshCmd.Flags().BoolVar(&sshInsecure, "insecure", false, "Disable SSH host key verification")
	sshCmd.Flags().StringVar(&sshUser, "user", "yeast", "SSH username")
	rootCmd.AddCommand(sshCmd)
}
