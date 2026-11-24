package main

import (
	"context"
	"fmt"
	"github.com/Twarga/yeast/internal/app"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newExecCmd(service *app.Service) *cobra.Command {
	var timeout time.Duration
	var cwd string
	var envPairs []string
	var user string
	var readStdin bool

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

			env, err := parseEnvPairs(envPairs)
			if err != nil {
				return err
			}

			options := app.ExecOptions{
				Timeout: timeout,
				Command: args[dash:],
				Cwd:     cwd,
				Env:     env,
				User:    user,
			}
			if dash == 1 {
				options.Target = args[0]
			}
			// Pipe stdin only when explicitly requested via --stdin or redirected from a regular file.
			// Never block forever on an open non-TTY pipe/socket (e.g. ssh, CI, subshells).
			shouldReadStdin := readStdin
			if !shouldReadStdin && !term.IsTerminal(int(os.Stdin.Fd())) {
				if fi, statErr := os.Stdin.Stat(); statErr == nil && fi.Mode().IsRegular() {
					shouldReadStdin = true
				}
			}
			if shouldReadStdin {
				data, err := io.ReadAll(os.Stdin)
				if err != nil {
					return fmt.Errorf("read stdin: %w", err)
				}
				if len(data) > 0 {
					options.Stdin = data
				}
			}

			result, err := service.Exec(context.Background(), options)
			if err != nil {
				return err
			}
			if err := renderExecCommandOutput(cmd.OutOrStdout(), cmd.ErrOrStderr(), result); err != nil {
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
	cmd.Flags().StringVar(&cwd, "cwd", "", "Working directory inside the guest")
	cmd.Flags().StringArrayVar(&envPairs, "env", nil, "Environment variable KEY=VAL (repeatable)")
	cmd.Flags().StringVar(&user, "user", "", "Guest user to run the command as (via sudo)")
	cmd.Flags().BoolVar(&readStdin, "stdin", false, "Read standard input and pipe it to the guest command")
	return cmd
}

func parseEnvPairs(pairs []string) (map[string]string, error) {
	if len(pairs) == 0 {
		return nil, nil
	}
	env := make(map[string]string, len(pairs))
	for _, pair := range pairs {
		key, val, ok := strings.Cut(pair, "=")
		if !ok || key == "" {
			return nil, fmt.Errorf("invalid --env %q, expected KEY=VAL", pair)
		}
		env[key] = val
	}
	return env, nil
}

func renderExecCommandOutput(stdout, stderr io.Writer, result app.ExecResult) error {
	if outputJSON {
		return renderCommandOutput(stdout, "exec", result)
	}
	if _, err := io.WriteString(stdout, result.Run.Stdout); err != nil {
		return err
	}
	if _, err := io.WriteString(stderr, result.Run.Stderr); err != nil {
		return err
	}
	if result.Run.Truncated && result.Run.FullOutputPath != "" {
		_, err := fmt.Fprintf(stderr, "\n(output truncated — full output at %s)\n", result.Run.FullOutputPath)
		return err
	}
	return nil
}
