package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"time"
	"yeast/internal/app"
	"yeast/internal/output"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var destroyPromptReader = func(r io.Reader) (string, error) {
	line, err := bufio.NewReader(r).ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return line, nil
}
var destroyTerminalCheck = writerIsTerminal

func newDestroyCmd(service *app.Service) *cobra.Command {
	var keepFiles bool
	var yes bool

	cmd := &cobra.Command{
		Use:   "destroy",
		Short: "Remove tracked VM runtime files for the current project",
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()
			resolvedKeepFiles, err := resolveDestroyKeepFilesMode(cmd, keepFiles, yes)
			if err != nil {
				return err
			}
			events, err := eventSink(cmd.OutOrStdout())
			if err != nil {
				return err
			}
			if events == nil && !outputQuiet && !outputJSON {
				events = output.NewProgressSink(cmd.ErrOrStderr())
			}
			result, err := service.Destroy(context.Background(), app.DestroyOptions{
				Events:    events,
				KeepFiles: resolvedKeepFiles,
			})
			if err != nil {
				return err
			}
			return renderCommandOutputWithTiming(cmd.OutOrStdout(), "destroy", result, time.Since(start))
		},
	}

	cmd.Flags().BoolVar(&keepFiles, "keep-files", false, "Stop instances but preserve runtime files")
	cmd.Flags().BoolVar(&yes, "yes", false, "Delete runtime files without prompting")

	return cmd
}

func resolveDestroyKeepFilesMode(cmd *cobra.Command, keepFiles, yes bool) (bool, error) {
	if keepFiles || yes || outputJSON {
		return keepFiles, nil
	}
	if !destroyTerminalCheck(cmd.InOrStdin()) || !destroyTerminalCheck(cmd.ErrOrStderr()) {
		return false, nil
	}

	fmt.Fprint(cmd.ErrOrStderr(), "Delete VM runtime files? [y/N]: ")
	line, err := destroyPromptReader(cmd.InOrStdin())
	if err != nil {
		return false, err
	}

	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return false, nil
	case "", "n", "no":
		return true, nil
	default:
		return false, fmt.Errorf("please answer y or n")
	}
}

func writerIsTerminal(value any) bool {
	file, ok := value.(interface{ Fd() uintptr })
	if !ok {
		return false
	}
	return term.IsTerminal(int(file.Fd()))
}
