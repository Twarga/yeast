package main

import (
	"fmt"
	"io"
	"strings"
	"yeast/internal/app"

	"github.com/spf13/cobra"
)

func newDoctorCmd(service *app.Service) *cobra.Command {
	var fix bool
	var fixYes bool

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check whether the host is ready for Yeast",
		Long: `Check the host for required dependencies and report blockers and warnings.

Use --fix to attempt guided remediation of fixable issues.
Use --fix --yes to run safe fixes non-interactively.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if fixYes && !fix {
				fix = true
			}
			if outputJSON && fix && !fixYes {
				return fmt.Errorf("--fix with --json requires --yes")
			}

			result, err := service.Doctor()
			if err != nil {
				return err
			}

			if !fix {
				if err := renderCommandOutput(cmd.OutOrStdout(), "doctor", result); err != nil {
					return err
				}
				if result.Blockers > 0 {
					return fmt.Errorf("doctor found %d blocker(s)", result.Blockers)
				}
				return nil
			}

			plan := app.BuildDoctorFixPlan(result)
			if !outputJSON {
				if err := renderCommandOutput(cmd.OutOrStdout(), "doctor", result); err != nil {
					return err
				}
				renderDoctorFixPlan(cmd.OutOrStdout(), plan)
			}

			if plan.Empty() {
				if outputJSON {
					if err := renderCommandOutput(cmd.OutOrStdout(), "doctor", result); err != nil {
						return err
					}
				}
				if result.Blockers > 0 {
					return fmt.Errorf("doctor found %d blocker(s)", result.Blockers)
				}
				return nil
			}

			if !fixYes {
				if !outputJSON {
					fmt.Fprintln(cmd.OutOrStdout(), "\n  Re-run with --fix --yes to apply the automatic fixes above.")
				}
				if result.Blockers > 0 {
					return fmt.Errorf("doctor found %d blocker(s)", result.Blockers)
				}
				return nil
			}

			applyResult, err := service.ApplyDoctorFixes(cmd.Context(), plan)
			if err != nil {
				return err
			}
			finalResult, err := service.Doctor()
			if err != nil {
				return err
			}

			if outputJSON {
				if err := renderCommandOutput(cmd.OutOrStdout(), "doctor", finalResult); err != nil {
					return err
				}
			} else {
				fmt.Fprintln(cmd.OutOrStdout())
				fmt.Fprintf(cmd.OutOrStdout(), "  Applied %d fix(es).\n", len(applyResult.Applied))
				if len(applyResult.SkippedManual) > 0 {
					fmt.Fprintf(cmd.OutOrStdout(), "  %d fix(es) still require manual work.\n", len(applyResult.SkippedManual))
				}
				fmt.Fprintln(cmd.OutOrStdout())
				if err := renderCommandOutput(cmd.OutOrStdout(), "doctor", finalResult); err != nil {
					return err
				}
			}

			result = finalResult
			if result.Blockers > 0 {
				return fmt.Errorf("doctor found %d blocker(s)", result.Blockers)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&fix, "fix", false, "Preview or apply resolvable issues")
	cmd.Flags().BoolVar(&fixYes, "yes", false, "Run safe fixes non-interactively (requires --fix)")

	return cmd
}

func renderDoctorFixPlan(w io.Writer, plan app.DoctorFixPlan) {
	if plan.Empty() {
		fmt.Fprintln(w, "\n  No automatic fixes are currently available.")
		return
	}

	fmt.Fprintf(w, "\n  Fix plan: %s\n", plan.Describe())
	for _, step := range plan.Steps {
		fmt.Fprintf(w, "  - %s: %s\n", step.CheckName, step.Fix.Description)
		if step.Fix.ManualOnly {
			if step.Fix.ManualInstructions != "" {
				fmt.Fprintf(w, "    manual: %s\n", step.Fix.ManualInstructions)
			}
			continue
		}
		if len(step.Fix.Command) > 0 {
			fmt.Fprintf(w, "    command: %s\n", joinCommand(step.Fix.Command, step.Fix.NeedsSudo))
		}
		if step.Fix.Path != "" {
			fmt.Fprintf(w, "    path: %s\n", step.Fix.Path)
		}
	}
}

func joinCommand(command []string, needsSudo bool) string {
	if len(command) == 0 {
		return ""
	}
	parts := append([]string{}, command...)
	if needsSudo {
		parts = append([]string{"sudo"}, parts...)
	}
	return strings.Join(parts, " ")
}
