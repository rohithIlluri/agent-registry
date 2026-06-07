package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/rohithilluri/agent-registry/internal/adapter"
	"github.com/rohithilluri/agent-registry/internal/config"
	"github.com/rohithilluri/agent-registry/internal/registry"
	"github.com/spf13/cobra"
)

func newRemoveCmd() *cobra.Command {
	var (
		agents  []string
		project bool
		yes     bool
	)
	cmd := &cobra.Command{
		Use:     "remove <name>",
		Aliases: []string{"uninstall", "rm"},
		Short:   "Uninstall an artifact",
		Args:    cobra.ExactArgs(1),
		Example: `  agr remove io.github.community/code-review
  agr remove filesystem --agent claude-code`,
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			scope := adapter.ScopeUser
			if project {
				scope = adapter.ScopeProject
			}

			// Determine targets.
			var targets []adapter.Adapter
			if len(agents) > 0 {
				for _, a := range agents {
					ad := adapter.ByName(a)
					if ad == nil {
						return fmt.Errorf("unknown agent %q", a)
					}
					targets = append(targets, ad)
				}
			} else {
				targets = adapter.Detect()
				if len(targets) == 0 {
					targets = adapter.All()
				}
			}

			if !yes {
				fmt.Printf("Remove %q from %s? [y/N] ", name, agentList(targets))
				var resp string
				_, _ = fmt.Scanln(&resp)
				if strings.ToLower(strings.TrimSpace(resp)) != "y" {
					return fmt.Errorf("cancelled")
				}
			}

			anyRemoved := false
			for _, t := range targets {
				if err := t.Remove(name, scope); err != nil {
					fmt.Fprintf(os.Stderr, "warning: %s: %v\n", t.Name(), err)
				} else {
					fmt.Printf("✓ Removed from %s\n", t.Name())
					anyRemoved = true
				}
			}
			if !anyRemoved {
				return fmt.Errorf("%q was not installed", name)
			}

			// Remove from installed DB.
			if err := removeFromDB(name, targets); err != nil {
				fmt.Fprintf(os.Stderr, "warning: update installed DB: %v\n", err)
			}
			return nil
		},
	}
	cmd.Flags().StringSliceVar(&agents, "agent", nil, "target agent(s)")
	cmd.Flags().BoolVar(&project, "project", false, "remove from project scope")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation")
	return cmd
}

func agentList(adapters []adapter.Adapter) string {
	names := make([]string, len(adapters))
	for i, a := range adapters {
		names[i] = a.Name()
	}
	return strings.Join(names, ", ")
}

func removeFromDB(name string, targets []adapter.Adapter) error {
	dbPath, err := config.InstalledDBPath()
	if err != nil {
		return err
	}
	data, err := os.ReadFile(dbPath) // #nosec G304 -- dbPath is ~/.agent-registry/installed.json
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var db registry.InstalledDB
	if err := json.Unmarshal(data, &db); err != nil {
		return err
	}
	agentNames := make(map[string]bool)
	for _, t := range targets {
		agentNames[t.Name()] = true
	}
	var kept []registry.InstalledEntry
	for _, e := range db.Entries {
		if e.Name == name && agentNames[e.Agent] {
			continue
		}
		kept = append(kept, e)
	}
	db.Entries = kept
	out, _ := json.MarshalIndent(db, "", "  ")
	return os.WriteFile(dbPath, out, 0o600)
}
