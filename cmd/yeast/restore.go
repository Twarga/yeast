package main

import (
	"context"
	"github.com/Twarga/yeast/internal/app"
	"github.com/Twarga/yeast/internal/ui"
	"github.com/Twarga/yeast/internal/ui/sprites"
	"time"

	"github.com/spf13/cobra"
)

func newRestoreCmd(service *app.Service) *cobra.Command {
	var all bool

	cmd := &cobra.Command{
		Use:   "restore <instance> <name> | --all <name>",
		Short: "Restore an instance disk from a snapshot, or every instance with --all",
		Args: func(cmd *cobra.Command, args []string) error {
			if all {
				return cobra.ExactArgs(1)(cmd, args)
			}
			return cobra.ExactArgs(2)(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()
			events, err := eventSink(cmd.OutOrStdout())
			if err != nil {
				return err
			}
			var stage *ui.LiveStage
			if events == nil {
				stage, events = newProgressStage(cmd.ErrOrStderr(), sprites.MoodWorking, ui.Voice.Restore())
				defer stageStop(stage)
			}
			if all {
				result, err := service.RestoreAll(context.Background(), app.RestoreAllOptions{
					Name:   args[0],
					Events: events,
				})
				stageFinish(stage, err)
				if err != nil {
					return err
				}
				return renderCommandOutputWithTiming(cmd.OutOrStdout(), "restore", result, time.Since(start))
			}
			result, err := service.Restore(context.Background(), app.RestoreOptions{
				Target: args[0],
				Name:   args[1],
				Events: events,
			})
			stageFinish(stage, err)
			if err != nil {
				return err
			}
			return renderCommandOutputWithTiming(cmd.OutOrStdout(), "restore", result, time.Since(start))
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "Restore every instance from one shared checkpoint name")
	return cmd
}
