package main

import (
	"context"
	"github.com/Twarga/yeast/internal/app"
	"github.com/Twarga/yeast/internal/ui"
	"github.com/Twarga/yeast/internal/ui/sprites"
	"time"

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
			var stage *ui.LiveStage
			if events == nil {
				stage, events = newProgressStage(cmd.ErrOrStderr(), sprites.MoodWorking, ui.Voice.DeleteSnapshot())
				defer stageStop(stage)
			}
			result, err := service.DeleteSnapshot(context.Background(), app.DeleteSnapshotOptions{
				Target: args[0],
				Name:   args[1],
				Events: events,
			})
			stageFinish(stage, err)
			if err != nil {
				return err
			}
			return renderCommandOutputWithTiming(cmd.OutOrStdout(), "delete-snapshot", result, time.Since(start))
		},
	}
}
