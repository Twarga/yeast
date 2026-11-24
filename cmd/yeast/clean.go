package main

import (
	"context"
	"github.com/Twarga/yeast/internal/app"
	"github.com/Twarga/yeast/internal/ui"
	"github.com/Twarga/yeast/internal/ui/sprites"
	"time"

	"github.com/spf13/cobra"
)

func newCleanCmd(service *app.Service) *cobra.Command {
	var timeout time.Duration
	var snapshots bool

	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Clean broken state and orphaned VM runtime files in the current project",
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()
			events, err := eventSink(cmd.OutOrStdout())
			if err != nil {
				return err
			}
			var stage *ui.LiveStage
			if events == nil {
				stage, events = newProgressStage(cmd.ErrOrStderr(), sprites.MoodWorking, ui.Voice.Clean())
				defer stageStop(stage)
			}
			if snapshots {
				result, err := service.CleanSnapshots(context.Background(), app.CleanOptions{
					Timeout: timeout,
					Events:  events,
				})
				stageFinish(stage, err)
				if err != nil {
					return err
				}
				return renderCommandOutputWithTiming(cmd.OutOrStdout(), "clean", result, time.Since(start))
			}
			result, err := service.Clean(context.Background(), app.CleanOptions{
				Timeout: timeout,
				Events:  events,
			})
			stageFinish(stage, err)
			if err != nil {
				return err
			}
			return renderCommandOutputWithTiming(cmd.OutOrStdout(), "clean", result, time.Since(start))
		},
	}

	cmd.Flags().DurationVar(&timeout, "timeout", 0, "Maximum time to wait for orphan cleanup")
	cmd.Flags().BoolVar(&snapshots, "snapshots", false, "Delete all snapshots for every instance (keeps running disks)")
	return cmd
}
