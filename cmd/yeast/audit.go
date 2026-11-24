package main

import (
	"github.com/spf13/cobra"

	"github.com/Twarga/yeast/internal/app"
	"github.com/Twarga/yeast/internal/audit"
)

func newAuditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "audit",
		Short: "Read the project audit log (.yeast/audit.jsonl)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			entries, err := audit.Read(".")
			if err != nil {
				return err
			}
			return renderCommandOutput(cmd.OutOrStdout(), "audit", app.AuditResult{
				Entries: entries,
			})
		},
	}
}
