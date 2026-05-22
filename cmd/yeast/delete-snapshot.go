package main

import (
	"context"
	"yeast/internal/app"

	"github.com/spf13/cobra"
)

func newDeleteSnapshotCmd(service *app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "delete-snapshot <instance> <name>",
		Short: "Delete a named snapshot for one instance",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := service.DeleteSnapshot(context.Background(), app.DeleteSnapshotOptions{
				Target: args[0],
				Name:   args[1],
			})
			if err != nil {
				return err
			}
			return renderCommandOutput(cmd.OutOrStdout(), "delete-snapshot", result)
		},
	}
}
