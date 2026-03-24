package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var outputJSON bool

var rootCmd = &cobra.Command{
	Use:           "yeast",
	Short:         "A local VM orchestrator",
	Long:          `Yeast is a tool for managing local virtual machines using KVM/QEMU.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		var reported *reportedJSONError
		if errors.As(err, &reported) {
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "%s %s\n", humanStyle("[fail]", ansiRed), err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&outputJSON, "json", false, "Output machine-readable JSON")
}
