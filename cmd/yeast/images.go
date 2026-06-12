package main

import (
	"fmt"
	"yeast/internal/app"

	"github.com/spf13/cobra"
)

func newImagesCmd(service *app.Service) *cobra.Command {
	var all bool
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "images",
		Short: "Manage cached VM images",
	}

	cleanCmd := &cobra.Command{
		Use:   "clean [image]",
		Short: "Remove cached VM images to free disk space",
		Args: func(cmd *cobra.Command, args []string) error {
			if all {
				if len(args) != 0 {
					return fmt.Errorf("--all does not accept an image name")
				}
				return nil
			}
			if len(args) != 1 {
				return fmt.Errorf("specify an image name or use --all")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			options := app.ImageCleanOptions{
				All:    all,
				DryRun: dryRun,
			}
			if len(args) > 0 {
				options.ImageName = args[0]
			}

			result, err := service.CleanImages(options)
			if err != nil {
				return err
			}
			return renderCommandOutput(cmd.OutOrStdout(), "images-clean", result)
		},
	}

	cleanCmd.Flags().BoolVar(&all, "all", false, "Remove all cached images")
	cleanCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be removed without removing")

	cmd.AddCommand(cleanCmd)
	return cmd
}
