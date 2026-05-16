package main

import (
	"context"
	"fmt"
	"yeast/internal/app"

	"github.com/spf13/cobra"
)

func newUpCmd(service *app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "up",
		Short: "Start the VMs described by the current project",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := service.Up(context.Background(), app.UpOptions{})
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			for _, instance := range result.Instances {
				fmt.Fprintf(out, "Started %s (%s)\n", instance.Name, instance.SSHAddress)
			}
			return nil
		},
	}
}
