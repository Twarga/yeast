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
			if err := renderCommandOutput(cmd.OutOrStdout(), "doctor", result); err != nil {
				return err
			}

			if result.Blockers > 0 {
				return fmt.Errorf("doctor found %d blocker(s)", result.Blockers)
			}
			return nil
		},
	}
}
