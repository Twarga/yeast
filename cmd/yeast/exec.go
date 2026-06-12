package main

import (
	"context"
	"fmt"
	"time"
	"yeast/internal/app"

	"github.com/spf13/cobra"
)

func newExecCmd(service *app.Service) *cobra.Command {
	var timeout time.Duration

	cmd := &cobra.Command{
		Use:   "exec [instance] -- <command...>",
		Short: "Run a command inside a running instance over SSH",
		RunE: func(cmd *cobra.Command, args []string) error {
			dash := cmd.ArgsLenAtDash()
			if dash < 0 {
				return fmt.Errorf("exec requires `-- <command...>`")
			}
			if dash > 1 {
				return fmt.Errorf("exec accepts at most one target instance before `--`")
			}
			if len(args) <= dash {
				return fmt.Errorf("exec command is required after `--`")
			}

			options := app.ExecOptions{
				Timeout: timeout,
				Command: args[dash:],
			}
			if dash == 1 {
				options.Target = args[0]
			}

			result, err := service.Exec(context.Background(), options)
			if err != nil {
				return err
			}
			if err := renderCommandOutput(cmd.OutOrStdout(), "exec", result); err != nil {
				return err
			}
			if result.Run.ExitCode != 0 {
				return commandExitError{code: result.Run.ExitCode}
			}
			if result.Run.TimedOut {
				return commandExitError{code: 124}
			}
			return nil
		},
	}

	cmd.Flags().DurationVar(&timeout, "timeout", 0, "Maximum time to wait for remote command completion")
	return cmd
}
