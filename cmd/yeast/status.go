package main

import (
	"context"
	"github.com/Twarga/yeast/internal/app"

	"github.com/spf13/cobra"
)

func newStatusCmd(service *app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show tracked VM status for the current project",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := service.Status(context.Background(), app.StatusOptions{})
			if err != nil {
				return err
			}
			renderStatusSprite(cmd.OutOrStdout(), result)
			return renderCommandOutput(cmd.OutOrStdout(), "status", result)
		},
	}
}
