package main

import (
	"fmt"
	"io"
	"yeast/internal/docs"

	"github.com/spf13/cobra"
)

func newDocsCmd() *cobra.Command {
	var list bool

	cmd := &cobra.Command{
		Use:   "docs [topic]",
		Short: "Render Yeast docs in the terminal",
		Args: func(cmd *cobra.Command, args []string) error {
			if list {
				if len(args) != 0 {
					return fmt.Errorf("--list does not accept a topic")
				}
				return nil
			}
			if len(args) > 1 {
				return fmt.Errorf("expected zero or one docs topic")
			}
			if len(args) == 1 && !docs.HasTopic(args[0]) {
				return fmt.Errorf("unknown docs topic %q", args[0])
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if outputJSON {
				return fmt.Errorf("docs command does not support --json")
			}
			if list {
				return writeDocsIndex(cmd.OutOrStdout())
			}

			topic := docs.DefaultTopic()
			if len(args) == 1 {
				topic = args[0]
			}
			return docs.Render(cmd.OutOrStdout(), topic)
		},
	}

	cmd.Flags().BoolVar(&list, "list", false, "List available docs topics")
	return cmd
}

func writeDocsIndex(w io.Writer) error {
	_, err := io.WriteString(w, docs.IndexMarkdown())
	return err
}
