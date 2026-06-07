package main

import (
	"context"
	"yeast/internal/app"

	"github.com/spf13/cobra"
)

func newSSHCmd(service *app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "ssh [instance] [-- <command...>]",
		Short: "Open SSH or run a command through SSH on a running instance",
		RunE: func(cmd *cobra.Command, args []string) error {
			dash := cmd.ArgsLenAtDash()
			options := app.SSHOptions{}
			switch {
			case dash == 0:
				options.RemoteCommand = args
			case dash == 1:
				options.Target = args[0]
				options.RemoteCommand = args[1:]
			case dash > 1:
				return cobra.MaximumNArgs(1)(cmd, args[:dash])
			case len(args) > 1:
				return cobra.MaximumNArgs(1)(cmd, args)
			case len(args) == 1:
				options.Target = args[0]
			}
			_, err := service.SSH(context.Background(), options)
			return err
		},
	}
}
