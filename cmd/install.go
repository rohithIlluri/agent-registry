package cmd

import (
	"fmt"
	"strings"

	"github.com/rohithilluri/agent-registry/internal/adapter"
	"github.com/rohithilluri/agent-registry/internal/config"
	"github.com/rohithilluri/agent-registry/internal/installer"
	"github.com/rohithilluri/agent-registry/internal/registry"
	"github.com/spf13/cobra"
)

func newInstallCmd() *cobra.Command {
	var (
		agents     []string
		global     bool
		project    bool
		yes        bool
		localIndex string
	)
	cmd := &cobra.Command{
		Use:   "install <name>",
		Short: "Install an artifact",
		Args:  cobra.ExactArgs(1),
		Example: `  agr install io.github.modelcontextprotocol/filesystem
  agr install io.github.community/code-review --agent claude-code
  agr install io.github.community/git-workflow --global --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			scope := adapter.ScopeUser
			if project {
				scope = adapter.ScopeProject
			}

			cfg, _ := config.Load()
			client := registry.NewClient()
			if localIndex != "" {
				client.LocalPath = localIndex
			} else if cfg != nil && cfg.IndexURL != "" {
				client.IndexURL = cfg.IndexURL
			}

			// Merge default-agent from config if not set on command line.
			if len(agents) == 0 && cfg != nil && cfg.DefaultAgent != "" {
				agents = []string{cfg.DefaultAgent}
			}
			// Merge auto-confirm from config.
			if !yes && cfg != nil && cfg.AutoConfirm {
				yes = true
			}

			// Resolve full name from index (support short names).
			idx, err := client.LoadIndex()
			if err != nil {
				return fmt.Errorf("load index: %w", err)
			}
			resolvedName, err := resolveName(idx, name)
			if err != nil {
				return err
			}

			art, err := client.LoadArtifact(resolvedName)
			if err != nil {
				return err
			}

			return installer.Install(art, installer.Options{
				Agents:      agents,
				Scope:       scope,
				AutoConfirm: yes,
			})
		},
	}
	cmd.Flags().StringSliceVar(&agents, "agent", nil, "target agent(s): claude-code, codex (default: all detected)")
	cmd.Flags().BoolVar(&global, "global", false, "install to user scope (default)")
	cmd.Flags().BoolVar(&project, "project", false, "install to current project scope")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation prompt")
	cmd.Flags().StringVar(&localIndex, "index", "", "use a local index file")
	return cmd
}

// resolveName looks up a possibly-short artifact name, returning the full name.
func resolveName(idx *registry.Index, query string) (string, error) {
	// Exact match first.
	if _, ok := registry.FindInIndex(idx, query); ok {
		return query, nil
	}
	// Suffix match: "filesystem" → "io.github.modelcontextprotocol/filesystem"
	var matches []registry.IndexEntry
	for _, e := range idx.Artifacts {
		short := e.Name
		if idx2 := strings.LastIndex(e.Name, "/"); idx2 >= 0 {
			short = e.Name[idx2+1:]
		}
		if short == query || strings.HasSuffix(e.Name, "/"+query) {
			matches = append(matches, e)
		}
	}
	if len(matches) == 1 {
		return matches[0].Name, nil
	}
	if len(matches) > 1 {
		names := make([]string, len(matches))
		for i, m := range matches {
			names[i] = m.Name
		}
		return "", fmt.Errorf("ambiguous name %q; did you mean one of:\n  %s", query, strings.Join(names, "\n  "))
	}
	return "", fmt.Errorf("artifact %q not found; run `agr search %s`", query, query)
}
