package main

import (
	"context"
	"time"
	"yeast/internal/app"
	"yeast/internal/output"

	"github.com/spf13/cobra"
)

func newRestoreCmd(service *app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "restore <instance> <name>",
		Short: "Restore a stopped instance disk from a named snapshot",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()
			events, err := eventSink(cmd.OutOrStdout())
			if err != nil {
				return err
			}
			if events == nil && !outputQuiet && !outputJSON {
				events = output.NewProgressSink(cmd.ErrOrStderr())
			}
			result, err := service.Restore(context.Background(), app.RestoreOptions{
				Target: args[0],
				Name:   args[1],
				Events: events,
			})
			if err != nil {
				return err
			}
			return renderCommandOutputWithTiming(cmd.OutOrStdout(), "restore", result, time.Since(start))
		},
	}
}
