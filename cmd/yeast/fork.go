package main

import (
	"context"
	"time"

	"github.com/Twarga/yeast/internal/app"
	"github.com/Twarga/yeast/internal/ui"
	"github.com/Twarga/yeast/internal/ui/sprites"

	"github.com/spf13/cobra"
)

func newForkCmd(service *app.Service) *cobra.Command {
	var count int
	var from string
	var boot bool

	cmd := &cobra.Command{
		Use:   "fork <instance>",
		Short: "Create copy-on-write sibling VMs from an instance or snapshot",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()
			events, err := eventSink(cmd.OutOrStdout())
			if err != nil {
				return err
			}
			var stage *ui.LiveStage
			if events == nil {
				stage, events = newProgressStage(cmd.ErrOrStderr(), sprites.MoodWorking, ui.Voice.Fork())
				defer stageStop(stage)
			}
			result, err := service.Fork(context.Background(), app.ForkOptions{
				Target: args[0],
				Count:  count,
				From:   from,
				Boot:   boot,
				Events: events,
			})
			stageFinish(stage, err)
			if err != nil {
				return err
			}
			return renderCommandOutputWithTiming(cmd.OutOrStdout(), "fork", result, time.Since(start))
		},
	}

	cmd.Flags().IntVar(&count, "count", 1, "Number of forked siblings to create")
	cmd.Flags().StringVar(&from, "from", "", "Fork from a named snapshot instead of the live disk")
	cmd.Flags().BoolVar(&boot, "boot", false, "Start the forks immediately after creation")
	return cmd
}
