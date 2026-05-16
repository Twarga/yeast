package main

import (
	"fmt"
	"os"
	"yeast/internal/app"

	"github.com/spf13/cobra"
)

var outputJSON bool

func newRootCmd(service *app.Service) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "yeast",
		Short:         "Linux-first local VM orchestration",
		Long:          "Yeast is the TwargaOps local VM engine for repeatable QEMU/KVM environments.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.PersistentFlags().BoolVar(&outputJSON, "json", false, "Output machine-readable JSON")
	cmd.AddCommand(newVersionCmd(service))
	return cmd
}

func Execute() {
	rootCmd := newRootCmd(app.NewService())
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func newVersionCmd(service *app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print Yeast version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(cmd.OutOrStdout(), service.Version())
		},
	}
}
