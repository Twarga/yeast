package main

import (
	"context"
	"yeast/internal/app"

	"github.com/spf13/cobra"
)

func newInspectCmd(service *app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "inspect <instance>",
		Short: "Show detailed state for one instance",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := service.Inspect(context.Background(), app.InspectOptions{
				GuestTargetOptions: app.GuestTargetOptions{Target: args[0]},
			})
			if err != nil {
				return err
			}
			return renderCommandOutput(cmd.OutOrStdout(), "inspect", result)
		},
	}
}
