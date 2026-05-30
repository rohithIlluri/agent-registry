package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/rohithilluri/agent-registry/internal/config"
	"github.com/rohithilluri/agent-registry/internal/registry"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	var agentFilter string
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List installed artifacts",
		Example: `  agr list
  agr list --agent claude-code`,
		RunE: func(cmd *cobra.Command, args []string) error {
			dbPath, err := config.InstalledDBPath()
			if err != nil {
				return err
			}
			data, err := os.ReadFile(dbPath)
			if os.IsNotExist(err) {
				fmt.Println("No artifacts installed.")
				return nil
			}
			if err != nil {
				return fmt.Errorf("read installed DB: %w", err)
			}
			var db registry.InstalledDB
			if err := json.Unmarshal(data, &db); err != nil {
				return fmt.Errorf("parse installed DB: %w", err)
			}
			if len(db.Entries) == 0 {
				fmt.Println("No artifacts installed.")
				return nil
			}
			bold := func(s string) string { return "\033[1m" + s + "\033[0m" }
			fmt.Printf("%-48s  %-14s  %-12s  %-8s  %s\n",
				bold("NAME"), bold("TYPE"), bold("AGENT"), bold("SCOPE"), bold("VERSION"))
			fmt.Println(strings.Repeat("─", 100))
			count := 0
			for _, e := range db.Entries {
				if agentFilter != "" && e.Agent != agentFilter {
					continue
				}
				fmt.Printf("%-48s  %-14s  %-12s  %-8s  %s\n",
					e.Name, e.Type, e.Agent, e.Scope, e.Version)
				count++
			}
			fmt.Printf("\n%d artifact(s) installed.\n", count)
			return nil
		},
	}
	cmd.Flags().StringVar(&agentFilter, "agent", "", "filter by agent (claude-code|codex)")
	return cmd
}
