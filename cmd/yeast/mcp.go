package main

import (
	"context"
	"os"

	"github.com/Twarga/yeast/internal/app"
	"github.com/Twarga/yeast/internal/mcp"

	"github.com/spf13/cobra"
)

func newMCPCmd(service *app.Service) *cobra.Command {
	var projectDir string

	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Run the Yeast MCP server over stdio (JSON-RPC 2.0)",
		Long: "Exposes vm_* tools to AI harnesses (Claude Code, Codex, OpenCode…). " +
			"Guardrails live in .yeast/agent-policy.yaml — MCP annotations are hints only.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root := projectDir
			if root == "" {
				root = "."
			}
			srv, err := mcp.NewServer(service, root)
			if err != nil {
				return err
			}
			return srv.ServeStdio(context.Background(), os.Stdin, os.Stdout)
		},
	}
	cmd.Flags().StringVar(&projectDir, "project", "", "Project directory (default: current directory)")
	return cmd
}
