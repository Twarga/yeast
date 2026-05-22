package main

import (
	"context"
	"yeast/internal/app"

	"github.com/spf13/cobra"
)

func newRestoreCmd(service *app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "restore <instance> <name>",
		Short: "Restore a stopped instance disk from a named snapshot",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := service.Restore(context.Background(), app.RestoreOptions{
				Target: args[0],
				Name:   args[1],
			})
			if err != nil {
				return err
			}
			return renderCommandOutput(cmd.OutOrStdout(), "restore", result)
		},
	}
}
