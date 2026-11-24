package main

import (
	"context"

	"github.com/Twarga/yeast/internal/app"
	"github.com/spf13/cobra"
)

func newScreenshotCmd(service *app.Service) *cobra.Command {
	var outputPath string
	cmd := &cobra.Command{
		Use:   "screenshot [instance]",
		Short: "Capture a PNG screenshot of a display-enabled instance via QMP",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := ""
			if len(args) == 1 {
				target = args[0]
			}
			result, err := service.Screenshot(context.Background(), app.ScreenshotOptions{
				Target:     target,
				OutputPath: outputPath,
			})
			if err != nil {
				return err
			}
			return renderCommandOutput(cmd.OutOrStdout(), "screenshot", result)
		},
	}
	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "output PNG path (default: project screenshots dir)")
	return cmd
}
