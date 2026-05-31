package adapter

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/rohithilluri/agent-registry/internal/registry"
)

// ClaudeCodeAdapter installs artifacts into Claude Code's ~/.claude/ hierarchy.
type ClaudeCodeAdapter struct{}

func NewClaudeCodeAdapter() *ClaudeCodeAdapter { return &ClaudeCodeAdapter{} }

func (a *ClaudeCodeAdapter) Name() string { return "claude-code" }

func (a *ClaudeCodeAdapter) Detect() bool {
	// Check for the ~/.claude directory or the `claude` binary on PATH.
	home, _ := os.UserHomeDir()
	if _, err := os.Stat(filepath.Join(home, ".claude")); err == nil {
		return true
	}
	_, err := exec.LookPath("claude")
	return err == nil
}

func (a *ClaudeCodeAdapter) InstallPath(t registry.ArtifactType, scope Scope) (string, error) {
	base, err := a.base(scope)
	if err != nil {
		return "", err
	}
	switch t {
	case registry.TypeSkill:
		return filepath.Join(base, "skills"), nil
	case registry.TypeSlashCommand:
		return filepath.Join(base, "commands"), nil
	case registry.TypeSubagent:
		return filepath.Join(base, "agents"), nil
	case registry.TypeHook:
		return filepath.Join(base, "hooks"), nil
	case registry.TypeMCPServer:
		if scope == ScopeProject {
			cwd, _ := os.Getwd()
			return cwd, nil // .mcp.json lives at project root
		}
		return base, nil // ~/.claude/settings.json
	default:
		return base, nil
	}
}

func (a *ClaudeCodeAdapter) Install(art *registry.Artifact, payload string, scope Scope) error {
	switch art.Type {
	case registry.TypeSkill:
		return a.installSkill(art, payload, scope)
	case registry.TypeMCPServer:
		return a.installMCP(art, scope)
	case registry.TypeSlashCommand:
		return a.installCommand(art, payload, scope)
	case registry.TypeSubagent:
		return a.installSubagent(art, payload, scope)
	default:
		return fmt.Errorf("claude-code: installing %q artifacts is not yet supported", art.Type)
	}
}

// installSkill copies the skill directory into ~/.claude/skills/<name>/.
func (a *ClaudeCodeAdapter) installSkill(art *registry.Artifact, payload string, scope Scope) error {
	base, err := a.base(scope)
	if err != nil {
		return err
	}
	dest := filepath.Join(base, "skills", shortName(art.Name))
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dest, err)
	}
	return copyDir(payload, dest)
}

// installMCP writes a server entry into .mcp.json (project) or ~/.claude/settings.json (user).
func (a *ClaudeCodeAdapter) installMCP(art *registry.Artifact, scope Scope) error {
	cfg, ok := art.Install["claude-code"]
	if !ok {
		cfg, ok = art.Install["any"]
	}
	if !ok || cfg.MCPServer == nil {
		return fmt.Errorf("artifact %q has no claude-code MCP install config", art.Name)
	}
	if scope == ScopeProject {
		return a.writeMCPJSON(art.Name, cfg.MCPServer)
	}
	return a.writeSettingsMCP(art.Name, cfg.MCPServer)
}

type mcpJSONFile struct {
	MCPServers map[string]registry.MCPServerConfig `json:"mcpServers"`
}

func (a *ClaudeCodeAdapter) writeMCPJSON(name string, srv *registry.MCPServerConfig) error {
	cwd, _ := os.Getwd()
	path := filepath.Join(cwd, ".mcp.json")
	var f mcpJSONFile
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &f)
	}
	if f.MCPServers == nil {
		f.MCPServers = make(map[string]registry.MCPServerConfig)
	}
	f.MCPServers[shortName(name)] = *srv
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal .mcp.json: %w", err)
	}
	return writeFileAtomic(path, data, 0o644) // project-level file; 0o644 is appropriate
}

type claudeSettings struct {
	MCPServers map[string]registry.MCPServerConfig `json:"mcpServers,omitempty"`
}

func (a *ClaudeCodeAdapter) writeSettingsMCP(name string, srv *registry.MCPServerConfig) error {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, ".claude", "settings.json")
	var s claudeSettings
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &s)
	}
	if s.MCPServers == nil {
		s.MCPServers = make(map[string]registry.MCPServerConfig)
	}
	s.MCPServers[shortName(name)] = *srv
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}
	return writeFileAtomic(path, data, 0o600)
}

func (a *ClaudeCodeAdapter) installCommand(art *registry.Artifact, payload string, scope Scope) error {
	base, err := a.base(scope)
	if err != nil {
		return err
	}
	dir := filepath.Join(base, "commands")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	// payload is the .md file, or we use the inline body
	cfg := art.Install["claude-code"]
	body := cfg.CommandBody
	if body == "" {
		if payload != "" {
			data, err := os.ReadFile(payload)
			if err != nil {
				return err
			}
			body = string(data)
		}
	}
	dest := filepath.Join(dir, shortName(art.Name)+".md")
	return os.WriteFile(dest, []byte(body), 0o644)
}

func (a *ClaudeCodeAdapter) installSubagent(art *registry.Artifact, payload string, scope Scope) error {
	base, err := a.base(scope)
	if err != nil {
		return err
	}
	dir := filepath.Join(base, "agents")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	cfg := art.Install["claude-code"]
	body := cfg.AgentBody
	if body == "" && payload != "" {
		data, err := os.ReadFile(payload)
		if err != nil {
			return err
		}
		body = string(data)
	}
	dest := filepath.Join(dir, shortName(art.Name)+".md")
	return os.WriteFile(dest, []byte(body), 0o644)
}

func (a *ClaudeCodeAdapter) IsInstalled(name string) (bool, error) {
	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(home, ".claude", "skills", shortName(name)),
		filepath.Join(home, ".claude", "commands", shortName(name)+".md"),
		filepath.Join(home, ".claude", "agents", shortName(name)+".md"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return true, nil
		}
	}
	// Check MCP in settings.json
	path := filepath.Join(home, ".claude", "settings.json")
	var s claudeSettings
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &s)
		if _, ok := s.MCPServers[shortName(name)]; ok {
			return true, nil
		}
	}
	return false, nil
}

func (a *ClaudeCodeAdapter) Remove(name string, scope Scope) error {
	base, err := a.base(scope)
	if err != nil {
		return err
	}
	short := shortName(name)
	candidates := []string{
		filepath.Join(base, "skills", short),
		filepath.Join(base, "commands", short+".md"),
		filepath.Join(base, "agents", short+".md"),
	}
	removed := false
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			if err := os.RemoveAll(p); err != nil {
				return err
			}
			removed = true
		}
	}
	// Remove from settings.json MCP
	if scope == ScopeUser {
		home, _ := os.UserHomeDir()
		path := filepath.Join(home, ".claude", "settings.json")
		var s claudeSettings
		if data, err := os.ReadFile(path); err == nil {
			_ = json.Unmarshal(data, &s)
			if _, ok := s.MCPServers[short]; ok {
				delete(s.MCPServers, short)
				if data, err := json.MarshalIndent(s, "", "  "); err == nil {
					_ = writeFileAtomic(path, data, 0o600)
				}
				removed = true
			}
		}
	}
	// Remove from project .mcp.json
	if scope == ScopeProject {
		cwd, _ := os.Getwd()
		path := filepath.Join(cwd, ".mcp.json")
		var f mcpJSONFile
		if data, err := os.ReadFile(path); err == nil {
			_ = json.Unmarshal(data, &f)
			if _, ok := f.MCPServers[short]; ok {
				delete(f.MCPServers, short)
				if data, err := json.MarshalIndent(f, "", "  "); err == nil {
					_ = writeFileAtomic(path, data, 0o644)
				}
				removed = true
			}
		}
	}
	if !removed {
		return fmt.Errorf("claude-code: %q not found (nothing to remove)", name)
	}
	return nil
}

func (a *ClaudeCodeAdapter) base(scope Scope) (string, error) {
	if scope == ScopeProject {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		return filepath.Join(cwd, ".claude"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude"), nil
}
