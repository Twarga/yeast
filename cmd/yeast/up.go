package main

import (
	"context"
	"yeast/internal/app"
	"yeast/internal/output"

	"github.com/spf13/cobra"
)

func newUpCmd(service *app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "up",
		Short: "Start the VMs described by the current project",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := service.Up(context.Background(), app.UpOptions{})
			if err != nil {
				return err
			}
			return output.RenderHuman(cmd.OutOrStdout(), "up", result)
		},
	}
}
