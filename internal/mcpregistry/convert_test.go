package mcpregistry

import (
	"testing"

	"github.com/rohithilluri/agent-registry/internal/registry"
)

func TestToArtifact_NPM(t *testing.T) {
	s := Server{
		ID:          "abc123",
		Name:        "io.github.modelcontextprotocol/filesystem",
		Description: "Read and write local files",
		Repository:  Repository{URL: "https://github.com/modelcontextprotocol/servers"},
		VersionDetail: VersionDetail{Version: "2.0.0"},
		Packages: []Package{
			{
				RegistryType: "npm",
				Identifier:   "@modelcontextprotocol/server-filesystem",
				Version:      "2.0.0",
				Transport:    "stdio",
			},
		},
	}

	art := ToArtifact(s)

	if art.Name != "io.github.modelcontextprotocol/filesystem" {
		t.Errorf("Name = %q", art.Name)
	}
	if art.Type != registry.TypeMCPServer {
		t.Errorf("Type = %q", art.Type)
	}
	if art.Version != "2.0.0" {
		t.Errorf("Version = %q", art.Version)
	}
	if art.Trust != registry.TrustCommunity {
		t.Errorf("Trust = %q (default should be community before sync overrides)", art.Trust)
	}
	cfg, ok := art.Install["any"]
	if !ok {
		t.Fatal("Install[any] missing")
	}
	if cfg.MCPServer == nil {
		t.Fatal("MCPServer nil")
	}
	if cfg.MCPServer.Command != "npx" {
		t.Errorf("Command = %q, want npx", cfg.MCPServer.Command)
	}
	if len(cfg.MCPServer.Args) < 2 || cfg.MCPServer.Args[1] != "@modelcontextprotocol/server-filesystem@2.0.0" {
		t.Errorf("Args = %v", cfg.MCPServer.Args)
	}
}

func TestToArtifact_PyPI(t *testing.T) {
	s := Server{
		Name:        "io.github.example/py-server",
		Description: "A Python MCP server",
		VersionDetail: VersionDetail{Version: "1.0.0"},
		Packages: []Package{
			{RegistryType: "pypi", Name: "example-mcp"},
		},
	}
	art := ToArtifact(s)
	cfg := art.Install["any"]
	if cfg.MCPServer == nil {
		t.Fatal("MCPServer nil")
	}
	if cfg.MCPServer.Command != "uvx" {
		t.Errorf("Command = %q, want uvx", cfg.MCPServer.Command)
	}
}

func TestToArtifact_NoPackages(t *testing.T) {
	s := Server{
		Name:        "io.github.example/empty",
		Description: "No packages",
		VersionDetail: VersionDetail{Version: "1.0.0"},
	}
	art := ToArtifact(s)
	if _, ok := art.Install["any"]; ok {
		t.Error("should not have install config when no packages")
	}
}

func TestToArtifact_FallbackVersion(t *testing.T) {
	s := Server{
		Name:    "io.github.example/no-version",
		Packages: []Package{{RegistryType: "npm", Name: "some-pkg"}},
	}
	art := ToArtifact(s)
	if art.Version != "latest" {
		t.Errorf("Version = %q, want latest", art.Version)
	}
}

func TestNormalizeName(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"io.github.user/server", "io.github.user/server"},
		{" /loose-name/ ", "loose-name"},
	}
	for _, c := range cases {
		got := normalizeName(c.in)
		if got != c.want {
			t.Errorf("normalizeName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
