package main

import (
	"context"
	"yeast/internal/app"

	"github.com/spf13/cobra"
)

func newDestroyCmd(service *app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "destroy",
		Short: "Remove tracked VM runtime files for the current project",
		RunE: func(cmd *cobra.Command, args []string) error {
			events, err := eventSink(cmd.OutOrStdout())
			if err != nil {
				return err
			}
			result, err := service.Destroy(context.Background(), app.DestroyOptions{Events: events})
			if err != nil {
				return err
			}
			return renderCommandOutput(cmd.OutOrStdout(), "destroy", result)
		},
	}
}
