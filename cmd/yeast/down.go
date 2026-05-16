package main

import (
	"context"
	"yeast/internal/app"
	"yeast/internal/output"

	"github.com/spf13/cobra"
)

func newDownCmd(service *app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "down",
		Short: "Stop running VMs in the current project",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := service.Down(context.Background(), app.DownOptions{})
			if err != nil {
				return err
			}
			return output.RenderHuman(cmd.OutOrStdout(), "down", result)
		},
	}
}
