package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/rohithilluri/agent-registry/internal/adapter"
	"github.com/spf13/cobra"
)

func newAgentsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "agents",
		Short: "Show supported agents, detection status, and install locations",
		Long: `List every agent agr can target, whether it was detected on this
system, and where each artifact type installs (user scope).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			bold := func(s string) string { return "\033[1m" + s + "\033[0m" }
			home, _ := os.UserHomeDir()

			for _, a := range adapter.All() {
				status := "\033[33mnot detected\033[0m"
				if a.Detect() {
					status = "\033[32m✓ detected\033[0m"
				}
				fmt.Printf("\n%s  %s\n", bold(a.Name()), status)
				for _, t := range a.SupportedTypes() {
					path, err := a.InstallPath(t, adapter.ScopeUser)
					if err != nil {
						continue
					}
					if home != "" {
						path = strings.Replace(path, home, "~", 1)
					}
					fmt.Printf("  %-14s %s\n", string(t), path)
				}
			}
			fmt.Println("\nUse --agent <name> on install/remove to target a specific agent.")
			return nil
		},
	}
}
