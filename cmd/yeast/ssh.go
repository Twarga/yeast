package main

import (
	"context"
	"yeast/internal/app"

	"github.com/spf13/cobra"
)

func newSSHCmd(service *app.Service) *cobra.Command {
	var verbose bool

	cmd := &cobra.Command{
		Use:   "ssh [instance]",
		Short: "Open an SSH session to a running instance",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 1 {
				return cobra.MaximumNArgs(1)(cmd, args)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			options := app.SSHOptions{Verbose: verbose}
			if len(args) == 1 {
				options.Target = args[0]
			}
			_, err := service.SSH(context.Background(), options)
			return err
		},
	}

	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show SSH connection details (passes -v to ssh)")

	return cmd
}
