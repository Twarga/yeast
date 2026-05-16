package main

import (
	"context"
	"fmt"
	"yeast/internal/app"

	"github.com/spf13/cobra"
)

func newDestroyCmd(service *app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "destroy",
		Short: "Remove tracked VM runtime files for the current project",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := service.Destroy(context.Background(), app.DestroyOptions{})
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			for _, instance := range result.Instances {
				fmt.Fprintf(out, "%s\t%s\n", instance.Name, instance.Status)
			}
			return nil
		},
	}
}
