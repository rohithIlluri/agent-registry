package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/rohithilluri/agent-registry/internal/registry"
	"github.com/spf13/cobra"
)

func newSearchCmd() *cobra.Command {
	var (
		typeFilter  string
		agentFilter string
		catFilter   string
		localIndex  string
	)
	cmd := &cobra.Command{
		Use:     "search [query]",
		Aliases: []string{"find"},
		Short:   "Search the registry",
		Example: `  agr search filesystem
  agr search --type mcp-server --agent claude-code
  agr search git --category devops`,
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.Join(args, " ")

			client := registry.NewClient()
			if localIndex != "" {
				client.LocalPath = localIndex
			}
			idx, err := client.LoadIndex()
			if err != nil {
				return fmt.Errorf("load index: %w", err)
			}

			results := registry.Search(idx, query, typeFilter, agentFilter, catFilter)
			if len(results) == 0 {
				fmt.Fprintln(os.Stderr, "No artifacts found.")
				return nil
			}

			printSearchResults(results)
			return nil
		},
	}
	cmd.Flags().StringVar(&typeFilter, "type", "", "filter by type (mcp-server|skill|subagent|slash-command|hook|plugin-bundle)")
	cmd.Flags().StringVar(&agentFilter, "agent", "", "filter by agent compatibility (claude-code|codex)")
	cmd.Flags().StringVar(&catFilter, "category", "", "filter by category")
	cmd.Flags().StringVar(&localIndex, "index", "", "use a local index file (for development)")
	return cmd
}

func printSearchResults(results []registry.IndexEntry) {
	bold := func(s string) string { return "\033[1m" + s + "\033[0m" }
	dim := func(s string) string { return "\033[2m" + s + "\033[0m" }
	trustColor := map[registry.TrustTier]string{
		registry.TrustCurated:   "\033[32m", // green
		registry.TrustVerified:  "\033[36m", // cyan
		registry.TrustCommunity: "\033[33m", // yellow
	}
	reset := "\033[0m"

	fmt.Printf("%-50s  %-14s  %-10s  %s\n",
		bold("NAME"), bold("TYPE"), bold("TRUST"), bold("DESCRIPTION"))
	fmt.Println(strings.Repeat("─", 110))

	for _, e := range results {
		trust := string(e.Trust)
		if c, ok := trustColor[e.Trust]; ok {
			trust = c + trust + reset
		}
		desc := e.Description
		if len(desc) > 60 {
			desc = desc[:57] + "…"
		}
		name := e.Name
		if len(name) > 48 {
			name = name[:45] + "…"
		}
		flags := ""
		if e.HasMCP {
			flags += " [mcp]"
		}
		if e.HasScripts {
			flags += " [scripts]"
		}
		if e.HasHooks {
			flags += " [hooks]"
		}
		fmt.Printf("%-50s  %-14s  %-10s  %s%s\n",
			bold(name), string(e.Type), trust, desc, dim(flags))
	}
	fmt.Printf("\n%d artifact(s) found.\n", len(results))
}
