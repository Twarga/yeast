package main

import (
	"context"
	"yeast/internal/app"

	"github.com/spf13/cobra"
)

func newProvisionCmd(service *app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "provision [instance]",
		Short: "Rerun provisioning for a reachable running instance",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 1 {
				return cobra.MaximumNArgs(1)(cmd, args)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			events, err := eventSink(cmd.OutOrStdout())
			if err != nil {
				return err
			}
			options := app.ProvisionOptions{}
			options.Events = events
			if len(args) == 1 {
				options.Target = args[0]
			}
			result, err := service.Provision(context.Background(), options)
			if err != nil {
				return err
			}
			return renderCommandOutput(cmd.OutOrStdout(), "provision", result)
		},
	}
}
