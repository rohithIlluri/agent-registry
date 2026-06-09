package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion <shell>",
		Short: "Generate shell completion script",
		Long: `Generate a shell completion script for agr.

Supported shells: bash, zsh, fish, powershell

Installation:

  # Bash (add to ~/.bashrc):
  source <(agr completion bash)

  # Zsh (add to ~/.zshrc):
  source <(agr completion zsh)
  # Or for persistent completions:
  agr completion zsh > "${fpath[1]}/_agr"

  # Fish:
  agr completion fish > ~/.config/fish/completions/agr.fish

  # PowerShell (add to $PROFILE):
  agr completion powershell | Out-String | Invoke-Expression`,
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		Args:      cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return rootCmd.GenBashCompletion(os.Stdout)
			case "zsh":
				return rootCmd.GenZshCompletion(os.Stdout)
			case "fish":
				return rootCmd.GenFishCompletion(os.Stdout, true)
			case "powershell":
				return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
			default:
				return fmt.Errorf("unsupported shell %q; choose: bash, zsh, fish, powershell", args[0])
			}
		},
	}
	return cmd
}
