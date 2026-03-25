package main

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
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
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if outputJSON {
			return jsonCommandError("ssh", "json_not_supported", fmt.Errorf("json output is not supported for interactive ssh sessions"))
		}

		lock, err := state.AcquireFileLock(stateFilePath, state.DefaultLockOptions())
		if err != nil {
			return fmt.Errorf("error acquiring state lock: %w", err)
		}
		defer releaseStateLock(lock)

		s, err := state.Load(stateFilePath)
		if err != nil {
			return fmt.Errorf("error loading state: %w", err)
		}
		s.Reconcile()

		name, inst, implicit, err := resolveSSHTarget(args, s)
		if err != nil {
			return err
		}
		if implicit {
			humanInfof("No instance specified; using %s", humanAccent(name))
			fmt.Println()
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

		humanSection("SSH Session")
		humanKeyValue("Instance", humanAccent(name))
		humanKeyValue("User", sshUser)
		humanKeyValue("Host", inst.IP)
		humanKeyValue("Port", fmt.Sprintf("%d", inst.SSHPort))
		if sshInsecure {
			humanWarnf("Host key verification is disabled for this connection")
		}
		fmt.Println()
		humanInfof("Opening SSH session")

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

func resolveSSHTarget(args []string, s *state.State) (string, state.Instance, bool, error) {
	if len(args) > 0 {
		name := args[0]
		inst, ok := s.Instances[name]
		if !ok || inst.Status != "running" {
			return "", state.Instance{}, false, fmt.Errorf("instance %s is not running", name)
		}
		return name, inst, false, nil
	}

	runningNames := make([]string, 0, len(s.Instances))
	for name, inst := range s.Instances {
		if inst.Status == "running" {
			runningNames = append(runningNames, name)
		}
	}
	sort.Strings(runningNames)

	switch len(runningNames) {
	case 0:
		return "", state.Instance{}, false, fmt.Errorf("no running instances found; run `yeast up` first or pass an instance name")
	case 1:
		name := runningNames[0]
		return name, s.Instances[name], true, nil
	default:
		return "", state.Instance{}, false, fmt.Errorf("multiple running instances found; choose one: %s", strings.Join(runningNames, ", "))
	}
}
