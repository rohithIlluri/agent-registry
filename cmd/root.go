package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const version = "0.1.0"

var rootCmd = &cobra.Command{
	Use:   "agr",
	Short: "Cross-agent registry CLI — install MCP servers, skills, and more",
	Long: `agr is a free, CLI-first registry for AI agent artifacts.

It discovers, installs, and manages MCP servers, SKILL.md skills,
slash commands, subagents, hooks, and plugin bundles across Claude Code
and OpenAI Codex (with more agents coming).

Index: https://github.com/rohithilluri/agent-registry
Docs : https://github.com/rohithilluri/agent-registry#readme`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute is the entry point called from main.go.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(
		newSearchCmd(),
		newInfoCmd(),
		newInstallCmd(),
		newListCmd(),
		newRemoveCmd(),
		newPublishCmd(),
		newInitCmd(),
		newVersionCmd(),
	)
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print agr version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("agr", version)
		},
	}
}
