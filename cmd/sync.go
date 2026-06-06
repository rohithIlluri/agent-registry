package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rohithilluri/agent-registry/internal/mcpregistry"
	"github.com/rohithilluri/agent-registry/internal/registry"
	"github.com/spf13/cobra"
)

func newRegistryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "registry",
		Short: "Manage the local registry index",
	}
	cmd.AddCommand(newRegistrySyncCmd())
	return cmd
}

func newRegistrySyncCmd() *cobra.Command {
	var (
		dryRun  bool
		outDir  string
		source  string
		trust   string
		limit   int
	)
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync MCP servers from the official MCP Registry into the local index",
		Long: `Fetch all published MCP servers from registry.modelcontextprotocol.io
and upsert them into the local registry/artifacts/ tree and registry/index.json.

Curated and verified entries already in the index are never downgraded.

Examples:
  agr registry sync
  agr registry sync --dry-run
  agr registry sync --out /tmp/my-registry
  agr registry sync --trust verified --limit 50`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSync(dryRun, outDir, source, trust, limit)
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print what would change without writing files")
	cmd.Flags().StringVar(&outDir, "out", "./registry", "output directory for registry files")
	cmd.Flags().StringVar(&source, "source", "", "override MCP Registry base URL")
	cmd.Flags().StringVar(&trust, "trust", "community", "trust tier to assign: community|verified|curated")
	cmd.Flags().IntVar(&limit, "limit", 0, "cap number of servers imported (0 = unlimited)")
	return cmd
}

func runSync(dryRun bool, outDir, source, trustStr string, limit int) error {
	tier := registry.TrustTier(trustStr)
	if tier != registry.TrustCommunity && tier != registry.TrustVerified && tier != registry.TrustCurated {
		return fmt.Errorf("invalid trust tier %q; choose: community, verified, curated", trustStr)
	}

	client := mcpregistry.NewClient()
	if source != "" {
		client.BaseURL = source
	}

	fmt.Fprintf(os.Stderr, "Fetching from %s ...\n", client.BaseURL)
	servers, err := client.ListAll(func(n int) {
		fmt.Fprintf(os.Stderr, "\r  fetched %d servers...", n)
	})
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return fmt.Errorf("list MCP servers: %w", err)
	}
	if limit > 0 && len(servers) > limit {
		servers = servers[:limit]
	}
	fmt.Fprintf(os.Stderr, "Converting %d servers...\n", len(servers))

	// Load existing index (if present) so we can merge.
	indexPath := filepath.Join(outDir, "index.json")
	existing := &registry.Index{Version: "1", Artifacts: nil}
	if data, err := os.ReadFile(indexPath); err == nil {
		_ = json.Unmarshal(data, existing)
	}

	// Build lookup of existing entries by name.
	existingMap := make(map[string]registry.IndexEntry, len(existing.Artifacts))
	for _, e := range existing.Artifacts {
		existingMap[e.Name] = e
	}

	var added, updated, skipped int
	artifacts := make([]registry.Artifact, 0, len(servers))
	for _, s := range servers {
		art := mcpregistry.ToArtifact(s)
		if art.Name == "" {
			skipped++
			continue
		}

		// Protect curated/verified entries from being downgraded.
		if prev, ok := existingMap[art.Name]; ok {
			if prev.Trust == registry.TrustCurated || prev.Trust == registry.TrustVerified {
				skipped++
				continue
			}
			art.Trust = tier
			updated++
		} else {
			art.Trust = tier
			added++
		}
		artifacts = append(artifacts, art)
	}

	fmt.Printf("Results: %d added, %d updated, %d skipped (protected/invalid)\n", added, updated, skipped)

	if dryRun {
		fmt.Println("--dry-run: no files written.")
		for _, art := range artifacts {
			fmt.Printf("  would write %s (%s)\n", art.Name, art.Version)
		}
		return nil
	}

	// Write per-artifact manifest files.
	artifactsDir := filepath.Join(outDir, "artifacts")
	for _, art := range artifacts {
		artPath := filepath.Join(artifactsDir, art.Name+".json")
		if err := os.MkdirAll(filepath.Dir(artPath), 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", filepath.Dir(artPath), err)
		}
		data, err := json.MarshalIndent(art, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal %s: %w", art.Name, err)
		}
		if err := os.WriteFile(artPath, data, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", artPath, err)
		}
	}

	// Merge new entries into the index, preserving protected ones.
	newEntries := mergeIndex(existing.Artifacts, artifacts)
	newIndex := registry.Index{
		Version:   existing.Version,
		Generated: time.Now().UTC().Format(time.RFC3339),
		Artifacts: newEntries,
	}
	if newIndex.Version == "" {
		newIndex.Version = "1"
	}
	indexData, err := json.MarshalIndent(newIndex, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal index: %w", err)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", outDir, err)
	}
	if err := os.WriteFile(indexPath, indexData, 0o644); err != nil {
		return fmt.Errorf("write index: %w", err)
	}

	fmt.Printf("Wrote %d entries to %s\n", len(newEntries), indexPath)
	return nil
}

// mergeIndex upserts newArts into the existing entries slice.
// Existing curated/verified entries are preserved as-is.
func mergeIndex(existing []registry.IndexEntry, newArts []registry.Artifact) []registry.IndexEntry {
	byName := make(map[string]int, len(existing))
	result := make([]registry.IndexEntry, len(existing))
	copy(result, existing)
	for i, e := range result {
		byName[e.Name] = i
	}

	for _, art := range newArts {
		entry := registry.IndexEntry{
			Name:        art.Name,
			DisplayName: art.DisplayName,
			Type:        art.Type,
			Version:     art.Version,
			Description: truncate(art.Description, 120),
			Category:    art.Category,
			Keywords:    art.Keywords,
			AgentCompat: art.AgentCompat,
			Trust:       art.Trust,
			HasMCP:      art.HasMCP,
			HasScripts:  art.HasScripts,
			HasHooks:    art.HasHooks,
		}
		if idx, ok := byName[art.Name]; ok {
			result[idx] = entry
		} else {
			byName[art.Name] = len(result)
			result = append(result, entry)
		}
	}
	return result
}

func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
