package main

import (
	"context"
	"fmt"
	"time"
	"yeast/internal/app"

	"github.com/spf13/cobra"
)

func newCopyCmd(service *app.Service) *cobra.Command {
	var toGuest bool
	var fromGuest bool
	var timeout time.Duration

	cmd := &cobra.Command{
		Use:   "copy <instance> [--to-guest|--from-guest] <source> <destination>",
		Short: "Copy a file between the host and a running instance",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			if toGuest == fromGuest {
				return fmt.Errorf("copy requires exactly one of --to-guest or --from-guest")
			}

			direction := app.CopyToGuest
			if fromGuest {
				direction = app.CopyFromGuest
			}

			result, err := service.Copy(context.Background(), app.CopyOptions{
				GuestTargetOptions: app.GuestTargetOptions{Target: args[0]},
				Direction:          direction,
				Source:             args[1],
				Destination:        args[2],
				Timeout:            timeout,
			})
			if err != nil {
				return err
			}
			return renderCommandOutput(cmd.OutOrStdout(), "copy", result)
		},
	}

	cmd.Flags().BoolVar(&toGuest, "to-guest", false, "Copy a local file to the guest")
	cmd.Flags().BoolVar(&fromGuest, "from-guest", false, "Copy a guest file to the local machine")
	cmd.Flags().DurationVar(&timeout, "timeout", 0, "Maximum time to wait for file transfer completion")
	return cmd
}
