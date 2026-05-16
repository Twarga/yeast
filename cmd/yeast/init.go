package main

import (
	"yeast/internal/app"
	"yeast/internal/output"

	"github.com/spf13/cobra"
)

func newInitCmd(service *app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize a Yeast project",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := service.Init(app.InitOptions{})
			if err != nil {
				return err
			}
			return output.RenderHuman(cmd.OutOrStdout(), "init", result)
		},
	}
}
