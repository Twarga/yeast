package main

import (
	"context"
	"time"
	"yeast/internal/app"
	"yeast/internal/output"

	"github.com/spf13/cobra"
)

func newSnapshotCmd(service *app.Service) *cobra.Command {
	var description string

	cmd := &cobra.Command{
		Use:   "snapshot <instance> <name>",
		Short: "Create a stopped-VM snapshot for one instance",
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
			result, err := service.Snapshot(context.Background(), app.SnapshotOptions{
				Target:      args[0],
				Name:        args[1],
				Description: description,
				Events:      events,
			})
			if err != nil {
				return err
			}
			return renderCommandOutputWithTiming(cmd.OutOrStdout(), "snapshot", result, time.Since(start))
		},
	}

	cmd.Flags().StringVar(&description, "description", "", "Optional snapshot description")
	return cmd
}
