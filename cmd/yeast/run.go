package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/Twarga/yeast/internal/app"
)

func newRunCmd(service *app.Service) *cobra.Command {
	var (
		rm      bool
		online  bool
		envs    []string
		memory  int
		cpus    int
		timeout time.Duration
	)
	cmd := &cobra.Command{
		Use:   "run --rm <image> -- <command...>",
		Short: "Run a command in an ephemeral, offline-by-default VM and destroy it",
		Long: "Boots a cached image into a throwaway VM (egress.mode=offline unless\n" +
			"--online is passed), executes the command over SSH, prints its output,\n" +
			"and destroys the VM — including on Ctrl-C.",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !rm {
				return fmt.Errorf("ephemeral run requires --rm (the VM is always destroyed after the command)")
			}
			image := args[0]
			var command []string
			if idx := cmd.ArgsLenAtDash(); idx >= 0 {
				command = args[idx:]
			} else if len(args) > 1 {
				command = args[1:]
			}
			if len(command) == 0 {
				return fmt.Errorf("missing command — use `yeast run --rm %s -- <cmd>`", image)
			}

			// Cancel on SIGINT/SIGTERM so the defer chain in Run tears the
			// VM down cleanly instead of leaking QEMU.
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			var stdin []byte
			if fi, err := os.Stdin.Stat(); err == nil && (fi.Mode()&os.ModeCharDevice) == 0 {
				data, _ := io.ReadAll(os.Stdin)
				stdin = data
			}

			result, err := service.Run(ctx, app.RunOptions{
				Image:     image,
				Command:   command,
				Env:       envs,
				MemoryMiB: memory,
				CPUs:      cpus,
				Timeout:   timeout,
				Stdin:     stdin,
				Online:    online,
			})
			if err != nil {
				return err
			}
			if renderErr := renderCommandOutput(cmd.OutOrStdout(), "run", result); renderErr != nil {
				return renderErr
			}
			if result.ExitCode != 0 {
				return commandExitError{code: result.ExitCode}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&rm, "rm", false, "Destroy the ephemeral VM after the command finishes (required)")
	cmd.Flags().BoolVar(&online, "online", false, "Allow outbound network access (default: offline)")
	cmd.Flags().StringArrayVar(&envs, "env", nil, "Environment variable KEY=VAL (repeatable)")
	cmd.Flags().IntVar(&memory, "memory", 0, "Memory in MiB (default 1024)")
	cmd.Flags().IntVar(&cpus, "cpus", 0, "vCPUs (default 2)")
	cmd.Flags().DurationVar(&timeout, "timeout", 0, "Command timeout (default 5m)")
	return cmd
}
