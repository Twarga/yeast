package main

import (
	"yeast/internal/app"

	"github.com/spf13/cobra"
)

func newInitCmd(service *app.Service) *cobra.Command {
	var templateName string
	var listTemplates bool
	var templates bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a Yeast project",
		RunE: func(cmd *cobra.Command, args []string) error {
			if listTemplates || templates {
				result, err := service.ListTemplates()
				if err != nil {
					return err
				}
				return renderCommandOutput(cmd.OutOrStdout(), "init", result)
			}

			result, err := service.Init(app.InitOptions{Template: templateName})
			if err != nil {
				return err
			}
			return renderCommandOutput(cmd.OutOrStdout(), "init", result)
		},
	}
	cmd.Flags().StringVar(&templateName, "template", "", "Initialize from a built-in template name or local template directory")
	cmd.Flags().BoolVar(&listTemplates, "list-templates", false, "List available built-in templates")
	cmd.Flags().BoolVar(&templates, "templates", false, "Alias for --list-templates")
	return cmd
}
