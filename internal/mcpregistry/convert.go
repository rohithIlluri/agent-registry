package mcpregistry

import (
	"strings"

	"github.com/rohithilluri/agent-registry/internal/registry"
)

// ToArtifact converts a Server from the upstream MCP Registry into our Artifact
// format. It uses the server's primary package to derive the install command.
func ToArtifact(s Server) registry.Artifact {
	name := normalizeName(s.Name)
	version := s.VersionDetail.Version
	if version == "" {
		version = "latest"
	}

	art := registry.Artifact{
		Name:        name,
		DisplayName: displayName(name),
		Type:        registry.TypeMCPServer,
		Version:     version,
		Description: s.Description,
		License:     "",
		Homepage:    s.Repository.URL,
		Repository:  s.Repository.URL,
		AgentCompat: []string{"claude-code", "codex"},
		Trust:       registry.TrustCommunity,
		Source:      registry.Source{Type: registry.SourceGit, Repo: repoPath(s.Repository.URL)},
		HasMCP:      true,
		Install:     registry.InstallConfig{},
	}

	if cfg := buildMCPConfig(s.Packages); cfg != nil {
		art.Install["any"] = registry.AgentInstallConfig{MCPServer: cfg}
	}

	return art
}

// normalizeName ensures the name uses reverse-DNS format.
// If the upstream name already looks like "io.github.user/pkg", keep it.
// Otherwise derive one from the repository URL.
func normalizeName(raw string) string {
	if strings.Contains(raw, ".") && strings.Contains(raw, "/") {
		return raw
	}
	// Strip leading/trailing slashes and spaces
	return strings.Trim(raw, "/ ")
}

func repoPath(repoURL string) string {
	repoURL = strings.TrimSuffix(repoURL, ".git")
	if idx := strings.Index(repoURL, "github.com/"); idx >= 0 {
		return repoURL[idx+len("github.com/"):]
	}
	return repoURL
}

func displayName(name string) string {
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		return name[idx+1:]
	}
	return name
}

// buildMCPConfig picks the best package from the server's package list and
// builds an MCPServerConfig for stdio transport.
func buildMCPConfig(pkgs []Package) *registry.MCPServerConfig {
	if len(pkgs) == 0 {
		return nil
	}
	// Prefer the first stdio package; fall back to the first entry.
	var chosen *Package
	for i := range pkgs {
		if pkgs[i].Transport == "stdio" || pkgs[i].Transport == "" {
			chosen = &pkgs[i]
			break
		}
	}
	if chosen == nil {
		chosen = &pkgs[0]
	}

	ident := chosen.Identifier
	if ident == "" {
		ident = chosen.Name
	}

	switch strings.ToLower(chosen.RegistryType) {
	case "npm":
		args := []string{"-y", ident}
		if chosen.Version != "" && chosen.Version != "latest" {
			args = []string{"-y", ident + "@" + chosen.Version}
		}
		return &registry.MCPServerConfig{Command: "npx", Args: args}
	case "pypi":
		return &registry.MCPServerConfig{
			Command: "uvx",
			Args:    []string{ident},
		}
	case "docker":
		return &registry.MCPServerConfig{
			Command: "docker",
			Args:    []string{"run", "--rm", "-i", ident},
		}
	default:
		if chosen.Command != "" {
			return &registry.MCPServerConfig{
				Command: chosen.Command,
				Args:    chosen.Args,
				Env:     chosen.Env,
			}
		}
		return nil
	}
}
