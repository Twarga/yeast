package main

import (
	"context"
	"yeast/internal/app"
	"yeast/internal/output"

	"github.com/spf13/cobra"
)

func newDestroyCmd(service *app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "destroy",
		Short: "Remove tracked VM runtime files for the current project",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := service.Destroy(context.Background(), app.DestroyOptions{})
			if err != nil {
				return err
			}
			return output.RenderHuman(cmd.OutOrStdout(), "destroy", result)
		},
	}
}
