package main

import (
	"fmt"
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

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Created %s\n", result.ConfigPath)
			fmt.Fprintf(out, "Created %s\n", result.MetadataPath)
			fmt.Fprintf(out, "Project ID: %s\n", result.ProjectID)
			return nil
		},
	}
}
