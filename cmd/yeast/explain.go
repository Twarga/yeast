package main

import (
	"fmt"
	"sort"

	"github.com/Twarga/yeast/internal/app"

	"github.com/spf13/cobra"
)

func newExplainCmd(service *app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "explain <error_code>",
		Short: "Explain a typed error code and its fix path",
		Long: `Print the offline knowledge card for an error code: what it means,
how to diagnose it, and the step-by-step fix. Codes appear in JSON error
envelopes (error.code) and in human error output.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				if outputJSON {
					return fmt.Errorf("explain requires an error code with --json")
				}
				w := cmd.OutOrStdout()
				codes := app.ExplainCodes()
				sort.Strings(codes)
				fmt.Fprintln(w, "Typed error codes:")
				for _, c := range codes {
					fmt.Fprintf(w, "  %s\n", c)
				}
				fmt.Fprintln(w, "\nUsage: yeast explain <code>")
				return nil
			}
			result, err := app.Explain(args[0])
			if err != nil {
				return err
			}
			return renderCommandOutput(cmd.OutOrStdout(), "explain", result)
		},
	}
}
