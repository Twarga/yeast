package main

import (
	"errors"
	"fmt"
	"yeast/internal/app"

	"github.com/spf13/cobra"
)

func newPullCmd(service *app.Service) *cobra.Command {
	var list bool

	cmd := &cobra.Command{
		Use:   "pull [image]",
		Short: "List or download trusted base images",
		Args: func(cmd *cobra.Command, args []string) error {
			if list {
				if len(args) != 0 {
					return fmt.Errorf("--list does not accept an image name")
				}
				return nil
			}
			if len(args) != 1 {
				return fmt.Errorf("expected exactly one image name")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			options := app.PullOptions{List: list}
			if !list {
				options.ImageName = args[0]
			}

			result, err := service.Pull(options)
			if err != nil {
				if errors.Is(err, app.ErrUnsupportedImage) {
					return fmt.Errorf("%w. use `yeast pull --list` to see supported images", err)
				}
				return err
			}

			out := cmd.OutOrStdout()
			if result.List {
				for _, image := range result.Images {
					fmt.Fprintln(out, image)
				}
				return nil
			}

			fmt.Fprintf(out, "Pulled %s\n", result.ImageName)
			fmt.Fprintf(out, "Saved %s\n", result.ImagePath)
			return nil
		},
	}

	cmd.Flags().BoolVar(&list, "list", false, "List supported images")
	return cmd
}
