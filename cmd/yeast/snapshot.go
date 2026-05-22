package main

import (
	"context"
	"yeast/internal/app"

	"github.com/spf13/cobra"
)

func newSnapshotCmd(service *app.Service) *cobra.Command {
	var description string

	cmd := &cobra.Command{
		Use:   "snapshot <instance> <name>",
		Short: "Create a stopped-VM snapshot for one instance",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := service.Snapshot(context.Background(), app.SnapshotOptions{
				Target:      args[0],
				Name:        args[1],
				Description: description,
			})
			if err != nil {
				return err
			}
			return renderCommandOutput(cmd.OutOrStdout(), "snapshot", result)
		},
	}

	cmd.Flags().StringVar(&description, "description", "", "Optional snapshot description")
	return cmd
}
