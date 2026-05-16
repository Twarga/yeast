package main

import (
	"yeast/internal/app"

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
			return renderCommandOutput(cmd.OutOrStdout(), "init", result)
		},
	}
}
