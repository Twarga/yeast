package main

import (
	"errors"
	"fmt"
	"time"
	"yeast/internal/app"

	"github.com/spf13/cobra"
)

func newPullCmd(service *app.Service) *cobra.Command {
	var list bool
	var cached bool

	cmd := &cobra.Command{
		Use:   "pull [image]",
		Short: "List, search, or download trusted base images",
		Args: func(cmd *cobra.Command, args []string) error {
			if list || cached {
				if len(args) != 0 {
					return fmt.Errorf("--list and --cached do not accept an image name")
				}
				return nil
			}
			if len(args) != 1 {
				return fmt.Errorf("expected exactly one image name")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()
			options := app.PullOptions{List: list, Cached: cached}
			if !list && !cached {
				options.ImageName = args[0]
			}

			result, err := service.Pull(options)
			if err != nil {
				if errors.Is(err, app.ErrUnsupportedImage) {
					return app.WrapError(app.ErrorCodeInvalidArgument, fmt.Sprintf("%v. use `yeast pull --list` to see supported images", err), err)
				}
				return err
			}
			return renderCommandOutputWithTiming(cmd.OutOrStdout(), "pull", result, time.Since(start))
		},
	}

	cmd.Flags().BoolVar(&list, "list", false, "List all supported images by category")
	cmd.Flags().BoolVar(&cached, "cached", false, "Show locally cached images")
	return cmd
}
