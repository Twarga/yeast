package main

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/Twarga/yeast/internal/app"
)

func newMigrateCmd(service *app.Service) *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Upgrade a v1 yeast project directory to v2 layout",
		Long: `Inspects the current project's .yeast state and upgrades it for the v2
engine: reconciles stale processes, backfills snapshot metadata, and creates
directories the new features expect (snapshots/, runtime dirs).

Run with --dry-run to preview changes without writing.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := service.Migrate(context.Background(), app.MigrateOptions{
				Root:   ".",
				DryRun: dryRun,
			})
			if err != nil {
				return err
			}
			return renderCommandOutput(cmd.OutOrStdout(), "migrate", *result)
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would change without writing")
	return cmd
}
