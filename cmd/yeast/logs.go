package main

import (
	"context"
	"github.com/Twarga/yeast/internal/app"

	"github.com/spf13/cobra"
)

func newLogsCmd(service *app.Service) *cobra.Command {
	var tailLines int
	var network bool

	cmd := &cobra.Command{
		Use:   "logs <instance>",
		Short: "Read the VM runtime log for one instance",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := service.Logs(context.Background(), app.LogsOptions{
				GuestTargetOptions: app.GuestTargetOptions{Target: args[0]},
				TailLines:          tailLines,
				Network:            network,
			})
			if err != nil {
				return err
			}
			return renderCommandOutput(cmd.OutOrStdout(), "logs", result)
		},
	}

	cmd.Flags().IntVar(&tailLines, "tail", 0, "Only return the last N lines")
	cmd.Flags().BoolVar(&network, "network", false, "Read the egress-proxy network decision log instead of the console log")
	return cmd
}
