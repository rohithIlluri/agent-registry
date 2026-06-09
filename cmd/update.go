package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/rohithilluri/agent-registry/internal/adapter"
	"github.com/rohithilluri/agent-registry/internal/config"
	"github.com/rohithilluri/agent-registry/internal/installer"
	"github.com/rohithilluri/agent-registry/internal/registry"
	"github.com/spf13/cobra"
)

func newUpdateCmd() *cobra.Command {
	var (
		agents     []string
		yes        bool
		localIndex string
	)
	cmd := &cobra.Command{
		Use:   "update [name]",
		Short: "Update installed artifacts to their latest registry versions",
		Example: `  agr update                              # update all installed artifacts
  agr update io.github.community/code-review
  agr update --agent claude-code --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			dbPath, err := config.InstalledDBPath()
			if err != nil {
				return err
			}
			data, err := os.ReadFile(dbPath) // #nosec G304 -- path from config.InstalledDBPath() under ~/.agent-registry/
			if os.IsNotExist(err) {
				fmt.Println("Nothing installed.")
				return nil
			}
			if err != nil {
				return fmt.Errorf("read installed DB: %w", err)
			}
			var db registry.InstalledDB
			if err := json.Unmarshal(data, &db); err != nil {
				return fmt.Errorf("parse installed DB: %w", err)
			}

			client := registry.NewClient()
			if localIndex != "" {
				client.LocalPath = localIndex
			}
			idx, err := client.LoadIndex()
			if err != nil {
				return fmt.Errorf("load index: %w", err)
			}

			// Build the set of names to update.
			toUpdate := make(map[string]bool)
			if len(args) > 0 {
				name, err := resolveName(idx, args[0])
				if err != nil {
					return err
				}
				toUpdate[name] = true
			} else {
				for _, e := range db.Entries {
					toUpdate[e.Name] = true
				}
			}

			if len(toUpdate) == 0 {
				fmt.Println("Nothing to update.")
				return nil
			}

			updated, skipped, failed := 0, 0, 0

			for name := range toUpdate {
				entry, ok := registry.FindInIndex(idx, name)
				if !ok {
					fmt.Fprintf(os.Stderr, "  skip %s: not found in registry\n", name)
					skipped++
					continue
				}

				// Find what version(s) are installed and which agents.
				var installedVersion string
				installedAgents := map[string]adapter.Scope{}
				for _, e := range db.Entries {
					if e.Name != name {
						continue
					}
					if len(agents) > 0 && !containsAgent(agents, e.Agent) {
						continue
					}
					installedVersion = e.Version
					installedAgents[e.Agent] = adapter.Scope(e.Scope)
				}

				if len(installedAgents) == 0 {
					skipped++
					continue
				}

				if installedVersion == entry.Version {
					fmt.Printf("  %-50s  already at %s\n", name, entry.Version)
					skipped++
					continue
				}

				fmt.Printf("  %-50s  %s → %s\n", name, installedVersion, entry.Version)

				art, err := client.LoadArtifact(name)
				if err != nil {
					fmt.Fprintf(os.Stderr, "  error loading %s: %v\n", name, err)
					failed++
					continue
				}

				agentNames := make([]string, 0, len(installedAgents))
				for a := range installedAgents {
					agentNames = append(agentNames, a)
				}

				// Use the scope from the first installed entry; all agents for this artifact.
				scope := adapter.ScopeUser
				for _, s := range installedAgents {
					scope = s
					break
				}

				if err := installer.Install(art, installer.Options{
					Agents:      agentNames,
					Scope:       scope,
					AutoConfirm: yes,
				}); err != nil {
					fmt.Fprintf(os.Stderr, "  error updating %s: %v\n", name, err)
					failed++
					continue
				}
				updated++
			}

			fmt.Printf("\nDone: %d updated, %d already current, %d failed.\n", updated, skipped, failed)
			if failed > 0 {
				return fmt.Errorf("%d artifact(s) failed to update", failed)
			}
			return nil
		},
	}
	cmd.Flags().StringSliceVar(&agents, "agent", nil, "limit update to specific agent(s)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation prompts")
	cmd.Flags().StringVar(&localIndex, "index", "", "use a local index file")
	return cmd
}

func containsAgent(agents []string, agent string) bool {
	for _, a := range agents {
		if a == agent {
			return true
		}
	}
	return false
}
