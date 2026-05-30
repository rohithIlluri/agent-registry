package cmd

import (
	"fmt"
	"strings"

	"github.com/rohithilluri/agent-registry/internal/registry"
	"github.com/spf13/cobra"
)

func newInfoCmd() *cobra.Command {
	var localIndex string
	cmd := &cobra.Command{
		Use:   "info <name>",
		Short: "Show full metadata for an artifact",
		Args:  cobra.ExactArgs(1),
		Example: `  agr info io.github.modelcontextprotocol/filesystem
  agr info io.github.community/code-review`,
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			client := registry.NewClient()
			if localIndex != "" {
				client.LocalPath = localIndex
			}

			// First check the index for a quick existence confirmation.
			idx, err := client.LoadIndex()
			if err != nil {
				return fmt.Errorf("load index: %w", err)
			}
			if _, ok := registry.FindInIndex(idx, name); !ok {
				// Try prefix match.
				var matches []registry.IndexEntry
				for _, e := range idx.Artifacts {
					if strings.Contains(e.Name, name) {
						matches = append(matches, e)
					}
				}
				if len(matches) == 0 {
					return fmt.Errorf("artifact %q not found; try `agr search %s`", name, name)
				}
				if len(matches) == 1 {
					name = matches[0].Name
				} else {
					fmt.Println("Multiple matches:")
					for _, m := range matches {
						fmt.Printf("  %s\n", m.Name)
					}
					return fmt.Errorf("be more specific")
				}
			}

			art, err := client.LoadArtifact(name)
			if err != nil {
				return err
			}
			printArtifactInfo(art)
			return nil
		},
	}
	cmd.Flags().StringVar(&localIndex, "index", "", "use a local index file")
	return cmd
}

func printArtifactInfo(art *registry.Artifact) {
	bold := func(s string) string { return "\033[1m" + s + "\033[0m" }
	label := func(k, v string) {
		if v == "" {
			return
		}
		fmt.Printf("  %-14s %s\n", bold(k+":"), v)
	}

	trustBadge := map[registry.TrustTier]string{
		registry.TrustCurated:   "\033[32m✔ curated\033[0m",
		registry.TrustVerified:  "\033[36m✔ verified\033[0m",
		registry.TrustCommunity: "\033[33m○ community\033[0m",
	}

	fmt.Println()
	fmt.Println(bold(art.Name))
	if art.DisplayName != "" && art.DisplayName != art.Name {
		fmt.Println(" ", art.DisplayName)
	}
	fmt.Println()

	label("Type", string(art.Type))
	label("Version", art.Version)
	label("Category", art.Category)
	label("Trust", trustBadge[art.Trust])
	label("License", art.License)
	if art.Author.Name != "" {
		author := art.Author.Name
		if art.Author.URL != "" {
			author += " <" + art.Author.URL + ">"
		}
		label("Author", author)
	}
	label("Homepage", art.Homepage)
	label("Repository", art.Repository)
	if len(art.Keywords) > 0 {
		label("Keywords", strings.Join(art.Keywords, ", "))
	}
	label("Agents", strings.Join(art.AgentCompat, ", "))
	if len(art.Permissions) > 0 {
		label("Permissions", strings.Join(art.Permissions, ", "))
	}
	if art.Checksum != "" {
		label("Checksum", art.Checksum)
	}

	fmt.Println()
	fmt.Println(bold("  Description"))
	fmt.Println("  " + art.Description)

	flags := []string{}
	if art.HasMCP {
		flags = append(flags, "\033[33m⚠ MCP server (external process)\033[0m")
	}
	if art.HasScripts {
		flags = append(flags, "\033[33m⚠ executable scripts\033[0m")
	}
	if art.HasHooks {
		flags = append(flags, "\033[33m⚠ lifecycle hooks (shell)\033[0m")
	}
	if len(flags) > 0 {
		fmt.Println()
		fmt.Println(bold("  Security flags"))
		for _, f := range flags {
			fmt.Println("  •", f)
		}
	}
	fmt.Println()
}
