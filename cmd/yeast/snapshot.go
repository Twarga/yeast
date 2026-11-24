package main

import (
	"context"
	"github.com/Twarga/yeast/internal/app"
	"github.com/Twarga/yeast/internal/ui"
	"github.com/Twarga/yeast/internal/ui/sprites"
	"time"

	"github.com/spf13/cobra"
)

func newSnapshotCmd(service *app.Service) *cobra.Command {
	var description string
	var all bool

	cmd := &cobra.Command{
		Use:   "snapshot <instance> <name> | --all <name>",
		Short: "Create a snapshot for one instance, or every instance with --all",
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
				stage, events = newProgressStage(cmd.ErrOrStderr(), sprites.MoodWorking, ui.Voice.Snapshot())
				defer stageStop(stage)
			}
			if all {
				result, err := service.SnapshotAll(context.Background(), app.SnapshotAllOptions{
					Name:        args[0],
					Description: description,
					Events:      events,
				})
				stageFinish(stage, err)
				if err != nil {
					return err
				}
				return renderCommandOutputWithTiming(cmd.OutOrStdout(), "snapshot", result, time.Since(start))
			}
			result, err := service.Snapshot(context.Background(), app.SnapshotOptions{
				Target:      args[0],
				Name:        args[1],
				Description: description,
				Events:      events,
			})
			stageFinish(stage, err)
			if err != nil {
				return err
			}
			return renderCommandOutputWithTiming(cmd.OutOrStdout(), "snapshot", result, time.Since(start))
		},
	}

	cmd.Flags().StringVar(&description, "description", "", "Optional snapshot description")
	cmd.Flags().BoolVar(&all, "all", false, "Snapshot every instance under one shared checkpoint name")
	return cmd
}
