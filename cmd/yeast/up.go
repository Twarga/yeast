package main

import (
	"context"
	"yeast/internal/app"

	"github.com/spf13/cobra"
)

func newUpCmd(service *app.Service) *cobra.Command {
	var noProvision bool
	cmd := &cobra.Command{
		Use:   "up",
		Short: "Start the VMs described by the current project",
		RunE: func(cmd *cobra.Command, args []string) error {
			events, err := commandEventSink(cmd.OutOrStdout(), cmd.ErrOrStderr())
			if err != nil {
				return err
			}
			result, err := service.Up(context.Background(), app.UpOptions{Events: events, SkipProvision: noProvision})
			if err != nil {
				return err
			}
			return renderCommandOutput(cmd.OutOrStdout(), "up", result)
		},
	}
	cmd.Flags().BoolVar(&noProvision, "no-provision", false, "Start VMs without waiting for cloud-init or running provisioning")
	return cmd
}
