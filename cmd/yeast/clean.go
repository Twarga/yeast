package main

import (
	"context"
	"time"
	"yeast/internal/app"
	"yeast/internal/output"

	"github.com/spf13/cobra"
)

func newCleanCmd(service *app.Service) *cobra.Command {
	var timeout time.Duration

	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Clean broken state and orphaned VM runtime files in the current project",
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()
			events, err := eventSink(cmd.OutOrStdout())
			if err != nil {
				return err
			}
			if events == nil && !outputQuiet && !outputJSON {
				events = output.NewProgressSink(cmd.ErrOrStderr())
			}
			result, err := service.Clean(context.Background(), app.CleanOptions{
				Timeout: timeout,
				Events:  events,
			})
			if err != nil {
				return err
			}
			return renderCommandOutputWithTiming(cmd.OutOrStdout(), "clean", result, time.Since(start))
		},
	}

	cmd.Flags().DurationVar(&timeout, "timeout", 0, "Maximum time to wait for orphan cleanup")
	return cmd
}
