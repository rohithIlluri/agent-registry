package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rohithilluri/agent-registry/internal/registry"
	"github.com/rohithilluri/agent-registry/internal/security"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newPublishCmd() *cobra.Command {
	var (
		name    string
		artType string
		outFile string
	)
	cmd := &cobra.Command{
		Use:   "publish [path]",
		Short: "Validate a local artifact and generate its registry manifest",
		Long: `publish validates your artifact and prints a manifest JSON suitable
for a pull request to the registry index.

For skills: point to a directory containing SKILL.md.
For MCP servers: point to a directory containing server.json.
For slash commands / subagents: point to a .md file.

Steps to publish:
  1. Run: agr publish ./my-skill > manifest.json
  2. Open a PR to: https://github.com/rohithilluri/agent-registry
  3. Add manifest.json to registry/artifacts/<namespace>/<name>.json
  4. Add an IndexEntry to registry/index.json`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "."
			if len(args) > 0 {
				path = args[0]
			}
			fi, err := os.Stat(path)
			if err != nil {
				return fmt.Errorf("cannot access %s: %w", path, err)
			}

			var art *registry.Artifact
			if fi.IsDir() {
				art, err = detectAndBuildManifest(path, name, artType)
			} else {
				art, err = buildFromFile(path, name, artType)
			}
			if err != nil {
				return err
			}

			out, err := json.MarshalIndent(art, "", "  ")
			if err != nil {
				return err
			}

			if outFile != "" {
				if err := os.WriteFile(outFile, out, 0o644); err != nil { // #nosec G306 -- user-specified output file for manifest
					return err
				}
				fmt.Fprintf(os.Stderr, "Manifest written to %s\n", outFile)
			} else {
				fmt.Println(string(out))
			}

			fmt.Fprintln(os.Stderr, "\n── Next steps ──────────────────────────────────────────")
			fmt.Fprintf(os.Stderr, "1. Fork https://github.com/rohithilluri/agent-registry\n")
			fmt.Fprintf(os.Stderr, "2. Add the manifest to registry/artifacts/%s.json\n", art.Name)
			fmt.Fprintf(os.Stderr, "3. Add an IndexEntry to registry/index.json\n")
			fmt.Fprintf(os.Stderr, "4. Open a pull request — CI will validate schema + scan.\n")
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "artifact name (reverse-DNS: io.github.<user>/<artifact>)")
	cmd.Flags().StringVar(&artType, "type", "", "artifact type override")
	cmd.Flags().StringVar(&outFile, "out", "", "write manifest to file instead of stdout")
	return cmd
}

// detectAndBuildManifest auto-detects artifact type from directory contents.
func detectAndBuildManifest(dir, name, artType string) (*registry.Artifact, error) {
	if artType == "" {
		if _, err := os.Stat(filepath.Join(dir, "SKILL.md")); err == nil {
			artType = string(registry.TypeSkill)
		} else if _, err := os.Stat(filepath.Join(dir, "server.json")); err == nil {
			artType = string(registry.TypeMCPServer)
		} else {
			return nil, fmt.Errorf("cannot detect artifact type; set --type explicitly")
		}
	}
	switch artType {
	case string(registry.TypeSkill):
		return buildSkillManifest(dir, name)
	case string(registry.TypeMCPServer):
		return buildMCPManifest(dir, name)
	default:
		return nil, fmt.Errorf("unsupported type %q for directory publish", artType)
	}
}

func buildFromFile(path, name, artType string) (*registry.Artifact, error) {
	if artType == "" {
		switch {
		case strings.HasSuffix(path, ".md"):
			artType = string(registry.TypeSlashCommand)
		default:
			return nil, fmt.Errorf("cannot detect artifact type from %s; set --type", path)
		}
	}
	data, err := os.ReadFile(path) // #nosec G304 -- path is CLI argument from developer publishing their own artifact
	if err != nil {
		return nil, err
	}
	cksum, err := security.SHA256File(path)
	if err != nil {
		return nil, fmt.Errorf("checksum %s: %w", path, err)
	}
	art := &registry.Artifact{
		Name:        name,
		Type:        registry.ArtifactType(artType),
		Version:     "1.0.0",
		Description: "TODO: add description",
		Category:    "productivity",
		AgentCompat: []string{"claude-code", "codex"},
		Trust:       registry.TrustCommunity,
		Checksum:    "",
		Source:      registry.Source{Type: registry.SourceGit},
		Install: registry.InstallConfig{
			"claude-code": {CommandBody: string(data)},
		},
	}
	art.Checksum = "sha256:" + cksum
	return art, nil
}

type skillFrontmatter struct {
	Name         string   `yaml:"name"`
	Description  string   `yaml:"description"`
	AllowedTools []string `yaml:"allowed-tools"`
	License      string   `yaml:"license"`
	Version      string   `yaml:"version"`
	Keywords     []string `yaml:"keywords"`
	Category     string   `yaml:"category"`
}

func buildSkillManifest(dir, nameOverride string) (*registry.Artifact, error) {
	skillPath := filepath.Join(dir, "SKILL.md")
	data, err := os.ReadFile(skillPath) // #nosec G304 -- path is developer's own artifact directory
	if err != nil {
		return nil, fmt.Errorf("read SKILL.md: %w", err)
	}
	fm, err := parseSkillFrontmatter(string(data))
	if err != nil {
		return nil, fmt.Errorf("parse SKILL.md frontmatter: %w", err)
	}
	if fm.Name == "" {
		return nil, fmt.Errorf("SKILL.md missing required frontmatter field: name")
	}
	if fm.Description == "" {
		return nil, fmt.Errorf("SKILL.md missing required frontmatter field: description")
	}

	artName := nameOverride
	if artName == "" {
		artName = "io.github.YOURUSER/" + fm.Name
	}

	hasScripts := false
	if _, err := os.Stat(filepath.Join(dir, "scripts")); err == nil {
		hasScripts = true
	}

	cksum, err := security.SHA256Dir(dir)
	if err != nil {
		return nil, fmt.Errorf("checksum %s: %w", dir, err)
	}
	cat := fm.Category
	if cat == "" {
		cat = "productivity"
	}
	ver := fm.Version
	if ver == "" {
		ver = "1.0.0"
	}

	return &registry.Artifact{
		Name:        artName,
		DisplayName: fm.Name,
		Type:        registry.TypeSkill,
		Version:     ver,
		Description: fm.Description,
		License:     fm.License,
		Keywords:    fm.Keywords,
		Category:    cat,
		AgentCompat: []string{"claude-code", "codex"},
		Trust:       registry.TrustCommunity,
		HasScripts:  hasScripts,
		Source:      registry.Source{Type: registry.SourceGit, Repo: "YOURUSER/YOURREPO"},
		Checksum:    "sha256:" + cksum,
		Install: registry.InstallConfig{
			"any": {
				SkillSource: "https://github.com/YOURUSER/YOURREPO/archive/refs/heads/main.tar.gz",
				SkillPath:   fm.Name,
			},
		},
	}, nil
}

func buildMCPManifest(dir, nameOverride string) (*registry.Artifact, error) {
	srvPath := filepath.Join(dir, "server.json")
	data, err := os.ReadFile(srvPath) // #nosec G304 -- path is developer's own artifact directory
	if err != nil {
		return nil, fmt.Errorf("read server.json: %w", err)
	}
	var srv map[string]interface{}
	if err := json.Unmarshal(data, &srv); err != nil {
		return nil, fmt.Errorf("parse server.json: %w", err)
	}

	srvName, _ := srv["name"].(string)
	desc, _ := srv["description"].(string)
	ver, _ := srv["version"].(string)
	if ver == "" {
		ver = "1.0.0"
	}

	artName := nameOverride
	if artName == "" && srvName != "" {
		artName = srvName
	}
	if artName == "" {
		artName = "io.github.YOURUSER/my-mcp-server"
	}

	cksum, err := security.SHA256File(srvPath)
	if err != nil {
		return nil, fmt.Errorf("checksum %s: %w", srvPath, err)
	}
	return &registry.Artifact{
		Name:        artName,
		Type:        registry.TypeMCPServer,
		Version:     ver,
		Description: desc,
		Category:    "productivity",
		AgentCompat: []string{"claude-code", "codex"},
		Trust:       registry.TrustCommunity,
		HasMCP:      true,
		Source:      registry.Source{Type: registry.SourceGit, Repo: "YOURUSER/YOURREPO"},
		Checksum:    "sha256:" + cksum,
		Install: registry.InstallConfig{
			"any": {
				MCPServer: &registry.MCPServerConfig{
					Command: "npx",
					Args:    []string{"-y", "YOURPACKAGE"},
				},
			},
		},
	}, nil
}

func parseSkillFrontmatter(content string) (*skillFrontmatter, error) {
	if !strings.HasPrefix(content, "---") {
		return nil, fmt.Errorf("SKILL.md must start with YAML frontmatter (---)")
	}
	end := strings.Index(content[3:], "---")
	if end < 0 {
		return nil, fmt.Errorf("SKILL.md frontmatter not closed with ---")
	}
	fmStr := content[3 : 3+end]
	var fm skillFrontmatter
	if err := yaml.Unmarshal([]byte(fmStr), &fm); err != nil {
		return nil, err
	}
	return &fm, nil
}
