package main

import (
	"fmt"
	"yeast/internal/app"

	"github.com/spf13/cobra"
)

func newDoctorCmd(service *app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check whether the host is ready for Yeast",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := service.Doctor()
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			for _, check := range result.Checks {
				fmt.Fprintf(out, "[%s] %s: %s\n", check.Status, check.Name, check.Details)
			}
			fmt.Fprintf(out, "Blockers: %d\n", result.Blockers)
			fmt.Fprintf(out, "Warnings: %d\n", result.Warnings)

			if result.Blockers > 0 {
				return fmt.Errorf("doctor found %d blocker(s)", result.Blockers)
			}
			return nil
		},
	}
}
