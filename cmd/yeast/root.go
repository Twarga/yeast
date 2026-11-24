package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/Twarga/yeast/internal/app"
	"github.com/Twarga/yeast/internal/ui"
	"github.com/Twarga/yeast/internal/ui/sprites"
	"github.com/spf13/cobra"
)

var outputJSON bool
var outputEvents bool
var outputQuiet bool
var outputPlain bool
var outputNoMotion bool
var outputQuirk bool

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
		RunE: func(cmd *cobra.Command, args []string) error {
			if renderHome(cmd.OutOrStdout(), service) {
				return nil
			}
			return cmd.Help()
		},
	}

	cmd.PersistentFlags().BoolVar(&outputJSON, "json", false, "Output machine-readable JSON")
	cmd.PersistentFlags().BoolVar(&outputEvents, "events", false, "Stream machine-readable lifecycle events as JSON Lines")
	cmd.PersistentFlags().BoolVarP(&outputQuiet, "quiet", "q", false, "Suppress progress output (final result only)")
	cmd.PersistentFlags().BoolVar(&outputPlain, "plain", false, "Plain human output without sprite art")
	cmd.PersistentFlags().BoolVar(&outputNoMotion, "no-motion", false, "Render the mascot as a still frame (reduced motion)")
	cmd.PersistentFlags().BoolVar(&outputQuirk, "quirk", false, "Easter egg: show the Yeast Cell quirk animation")
	_ = cmd.PersistentFlags().MarkHidden("quirk")
	cmd.AddCommand(newCleanCmd(service))
	cmd.AddCommand(newCompletionCmd())
	cmd.AddCommand(newDocsCmd())
	cmd.AddCommand(newDaemonCmd(service))
	cmd.AddCommand(newDownCmd(service))
	cmd.AddCommand(newEgressProxyCmd())
	cmd.AddCommand(newAuditCmd())
	cmd.AddCommand(newRunCmd(service))
	cmd.AddCommand(newExportCmd(service))
	cmd.AddCommand(newBootISOCmd(service))
	cmd.AddCommand(newDeleteSnapshotCmd(service))
	cmd.AddCommand(newDestroyCmd(service))
	cmd.AddCommand(newImagesCmd(service))
	cmd.AddCommand(newInitCmd(service))
	cmd.AddCommand(newDoctorCmd(service))
	cmd.AddCommand(newExecCmd(service))
	cmd.AddCommand(newExplainCmd(service))
	cmd.AddCommand(newForkCmd(service))
	cmd.AddCommand(newCopyCmd(service))
	cmd.AddCommand(newInspectCmd(service))
	cmd.AddCommand(newInputCmd(service))
	cmd.AddCommand(newLogsCmd(service))
	cmd.AddCommand(newMigrateCmd(service))
	cmd.AddCommand(newMCPCmd(service))
	cmd.AddCommand(newPullCmd(service))
	cmd.AddCommand(newProvisionCmd(service))
	cmd.AddCommand(newRestoreCmd(service))
	cmd.AddCommand(newScreenshotCmd(service))
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
		if !outputPlain && !outputQuiet && !outputEvents && ui.StylingEnabled(os.Stderr) {
			renderStyledError(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		if hint := app.ErrorHint(err); hint != "" {
			fmt.Fprintf(os.Stderr, "\n  Hint: %s\n", hint)
		}
		os.Exit(1)
	}
}

func renderStyledError(w *os.File, err error) {
	theme := ui.NewTheme(w)
	what := err.Error()
	var code string
	var appErr *app.AppError
	if errors.As(err, &appErr) && appErr.Code != "" {
		code = string(appErr.Code)
	}
	hint := app.ErrorHint(err)
	card := theme.ErrorCard(what, "", hint, code)
	depth := ui.DepthFor(w)
	if depth != sprites.DepthNone && ui.TierFor(ui.WidthFor(w)) == ui.WidthFull {
		sprite := strings.Join(sprites.RenderStill(sprites.MoodError, depth), "\n")
		fmt.Fprintln(w, lipgloss.JoinHorizontal(lipgloss.Center, sprite, "   ", card))
		return
	}
	fmt.Fprintln(w, card)
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
