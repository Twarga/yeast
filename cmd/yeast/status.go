package main

import (
	"context"
	"yeast/internal/app"
	"yeast/internal/output"

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
			return output.RenderHuman(cmd.OutOrStdout(), "status", result)
		},
	}
}
