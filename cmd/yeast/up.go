package main

import (
	"context"
	"fmt"
	"os"
	"time"
	"yeast/internal/app"
	"yeast/internal/output"

	"github.com/spf13/cobra"
)

func newUpCmd(service *app.Service) *cobra.Command {
	var noProvision bool
	var reprovision bool
	var sequential bool
	var profile bool

	cmd := &cobra.Command{
		Use:   "up",
		Short: "Start the VMs described by the current project",
		RunE: func(cmd *cobra.Command, args []string) error {
			if noProvision && reprovision {
				return fmt.Errorf("--no-provision and --reprovision cannot be used together")
			}
			start := time.Now()
			w := cmd.OutOrStdout()
			events, err := eventSink(w)
			if err != nil {
				return err
			}
			if events == nil && !outputQuiet && !outputJSON {
				events = output.NewProgressSink(cmd.ErrOrStderr())
			}
			result, err := service.Up(context.Background(), app.UpOptions{
				Events:      events,
				NoProvision: noProvision,
				Reprovision: reprovision,
				Sequential:  sequential,
				Profile:     profile,
			})
			if err != nil {
				return err
			}
			if renderErr := renderCommandOutputWithTiming(w, "up", result, time.Since(start)); renderErr != nil {
				return renderErr
			}
			if profile && result.Profile != nil {
				profileResult := &output.ProfileResult{}
				for _, p := range result.Profile.Phases {
					profileResult.Phases = append(profileResult.Phases, output.ProfilePhase{
						Name:     p.Name,
						Started:  time.Now().Add(-p.Duration),
						Finished: time.Now(),
					})
				}
				profileResult.Total = result.Profile.Total
				output.RenderProfile(os.Stderr, profileResult)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&noProvision, "no-provision", false, "Skip provisioning even on first boot")
	cmd.Flags().BoolVar(&reprovision, "reprovision", false, "Force re-provision even if already provisioned")
	cmd.Flags().BoolVar(&sequential, "sequential", false, "Boot VMs sequentially (for debugging)")
	cmd.Flags().BoolVar(&profile, "profile", false, "Show boot-time profiling breakdown")

	return cmd
}
