package main

import (
	"errors"
	"fmt"
	"os"
	"yeast/internal/app"

	"github.com/spf13/cobra"
)

var outputJSON bool
var outputEvents bool
var outputQuiet bool

type commandExitError struct {
	code int
}

func (e commandExitError) Error() string {
	return fmt.Sprintf("command exited with status %d", e.code)
}

func (e commandExitError) ExitCode() int {
	if e.code <= 0 {
		return 1
	}
	if e.code > 255 {
		return 255
	}
	return e.code
}

func newRootCmd(service *app.Service) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "yeast",
		Short:         "Linux-first local VM orchestration",
		Long:          "Yeast is the TwargaOps local VM engine for repeatable QEMU/KVM environments.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.PersistentFlags().BoolVar(&outputJSON, "json", false, "Output machine-readable JSON")
	cmd.PersistentFlags().BoolVar(&outputEvents, "events", false, "Stream machine-readable lifecycle events as JSON Lines")
	cmd.PersistentFlags().BoolVarP(&outputQuiet, "quiet", "q", false, "Suppress progress output (final result only)")
	cmd.AddCommand(newCleanCmd(service))
	cmd.AddCommand(newCompletionCmd())
	cmd.AddCommand(newDocsCmd())
	cmd.AddCommand(newDownCmd(service))
	cmd.AddCommand(newDeleteSnapshotCmd(service))
	cmd.AddCommand(newDestroyCmd(service))
	cmd.AddCommand(newImagesCmd(service))
	cmd.AddCommand(newInitCmd(service))
	cmd.AddCommand(newDoctorCmd(service))
	cmd.AddCommand(newExecCmd(service))
	cmd.AddCommand(newCopyCmd(service))
	cmd.AddCommand(newInspectCmd(service))
	cmd.AddCommand(newLogsCmd(service))
	cmd.AddCommand(newPullCmd(service))
	cmd.AddCommand(newProvisionCmd(service))
	cmd.AddCommand(newRestoreCmd(service))
	cmd.AddCommand(newSnapshotCmd(service))
	cmd.AddCommand(newSnapshotsCmd(service))
	cmd.AddCommand(newSSHCmd(service))
	cmd.AddCommand(newStatusCmd(service))
	cmd.AddCommand(newUpdateCmd())
	cmd.AddCommand(newUpCmd(service))
	cmd.AddCommand(newVersionCmd(service))
	return cmd
}

func Execute() {
	rootCmd := newRootCmd(app.NewService())
	if err := rootCmd.Execute(); err != nil {
		var exitErr commandExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}
		if outputJSON {
			if renderErr := renderCommandError(os.Stdout, err); renderErr == nil {
				os.Exit(1)
			}
		}
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		if hint := app.ErrorHint(err); hint != "" {
			fmt.Fprintf(os.Stderr, "\n  Hint: %s\n", hint)
		}
		os.Exit(1)
	}
}

func newVersionCmd(service *app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print Yeast version",
		Run: func(cmd *cobra.Command, args []string) {
			_ = renderCommandOutput(cmd.OutOrStdout(), "version", service.Version())
		},
	}
}
