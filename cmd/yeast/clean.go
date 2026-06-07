package main

import (
	"context"
	"yeast/internal/app"

	"github.com/spf13/cobra"
)

func newCleanCmd(service *app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "clean",
		Short: "Force clean tracked and orphaned VMs in the current project",
		RunE: func(cmd *cobra.Command, args []string) error {
			events, err := commandEventSink(cmd.OutOrStdout(), cmd.ErrOrStderr())
			if err != nil {
				return err
			}
			result, err := service.Clean(context.Background(), app.CleanOptions{Events: events})
			if err != nil {
				return err
			}
			return renderCommandOutput(cmd.OutOrStdout(), "clean", result)
		},
	}
}
