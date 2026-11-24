package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/Twarga/yeast/internal/app"
	"github.com/spf13/cobra"
)

func newInputCmd(service *app.Service) *cobra.Command {
	var keyChord string
	var text string
	var holdMs int
	cmd := &cobra.Command{
		Use:   "input [instance]",
		Short: "Send keystrokes or typed text to a display-enabled instance via QMP",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := ""
			if len(args) == 1 {
				target = args[0]
			}
			var keys []string
			if keyChord != "" {
				for _, k := range strings.Split(keyChord, "+") {
					k = strings.TrimSpace(k)
					if k == "" {
						return fmt.Errorf("invalid --key chord %q (empty key name)", keyChord)
					}
					keys = append(keys, k)
				}
			}
			result, err := service.Input(context.Background(), app.InputOptions{
				Target:     target,
				Keys:       keys,
				Text:       text,
				HoldTimeMs: holdMs,
			})
			if err != nil {
				return err
			}
			return renderCommandOutput(cmd.OutOrStdout(), "input", result)
		},
	}
	cmd.Flags().StringVar(&keyChord, "key", "", "simultaneous key chord, e.g. meta_l+ret or ctrl+alt+f2")
	cmd.Flags().StringVar(&text, "text", "", "type ASCII text (e.g. --text 'echo hi\\n')")
	cmd.Flags().IntVar(&holdMs, "hold", 80, "hold time between key press and release (ms)")
	return cmd
}
