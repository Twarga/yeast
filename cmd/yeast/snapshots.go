package main

import (
	"context"
	"yeast/internal/app"

	"github.com/spf13/cobra"
)

func newSnapshotsCmd(service *app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "snapshots <instance>",
		Short: "List snapshots for one instance",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := service.Snapshots(context.Background(), app.SnapshotsOptions{
				Target: args[0],
			})
			if err != nil {
				return err
			}
			return renderCommandOutput(cmd.OutOrStdout(), "snapshots", result)
		},
	}
}
