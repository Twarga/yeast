package main

import (
	"context"
	"fmt"
	"yeast/internal/app"

	"github.com/spf13/cobra"
)

func newStatusCmd(service *app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show tracked VM status for the current project",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := service.Status(context.Background(), app.StatusOptions{})
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			for _, instance := range result.Instances {
				if instance.SSHPort > 0 {
					fmt.Fprintf(out, "%s\t%s\t127.0.0.1:%d\n", instance.Name, instance.Status, instance.SSHPort)
					continue
				}
				fmt.Fprintf(out, "%s\t%s\n", instance.Name, instance.Status)
			}
			return nil
		},
	}
}
