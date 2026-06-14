package main

import (
	"context"
	"time"
	"yeast/internal/app"
	"yeast/internal/output"

	"github.com/spf13/cobra"
)

func newDownCmd(service *app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "down",
		Short: "Stop running VMs in the current project",
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()
			events, err := eventSink(cmd.OutOrStdout())
			if err != nil {
				return err
			}
			if events == nil && !outputQuiet && !outputJSON {
				events = output.NewProgressSink(cmd.ErrOrStderr())
			}
			result, err := service.Down(context.Background(), app.DownOptions{Events: events})
			if err != nil {
				return err
			}
			return renderCommandOutputWithTiming(cmd.OutOrStdout(), "down", result, time.Since(start))
		},
	}
}
