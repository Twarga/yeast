package main

import (
	"context"
	"time"
	"yeast/internal/app"
	"yeast/internal/output"

	"github.com/spf13/cobra"
)

func newDeleteSnapshotCmd(service *app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "delete-snapshot <instance> <name>",
		Short: "Delete a named snapshot for one instance",
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
			result, err := service.DeleteSnapshot(context.Background(), app.DeleteSnapshotOptions{
				Target: args[0],
				Name:   args[1],
				Events: events,
			})
			if err != nil {
				return err
			}
			return renderCommandOutputWithTiming(cmd.OutOrStdout(), "delete-snapshot", result, time.Since(start))
		},
	}
}
