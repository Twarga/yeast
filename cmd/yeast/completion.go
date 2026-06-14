package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newCompletionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Long: `Generate completion scripts for your shell.

To load completions:

Bash:
  $ source <(yeast completion bash)
  # To load completions for each session, execute once:
  # Linux:
  $ yeast completion bash > /etc/bash_completion.d/yeast
  # macOS:
  $ yeast completion bash > $(brew --prefix)/etc/bash_completion.d/yeast

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc
  # To load completions for each session, execute once:
  $ yeast completion zsh > "${fpath[1]}/_yeast"
  # You will need to start a new shell for this setup to take effect.

Fish:
  $ yeast completion fish | source
  # To load completions for each session, execute once:
  $ yeast completion fish > ~/.config/fish/completions/yeast.fish

PowerShell:
  PS> yeast completion powershell | Out-String | Invoke-Expression
  # To load completions for every new session, run:
  PS> yeast completion powershell > yeast.ps1
  # and source this file from your PowerShell profile.
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletionV2(os.Stdout, true)
			case "zsh":
				return cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				return cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			default:
				return fmt.Errorf("unsupported shell: %s", args[0])
			}
		},
	}
}
