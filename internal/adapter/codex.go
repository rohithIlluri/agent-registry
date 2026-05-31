package adapter

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/rohithilluri/agent-registry/internal/registry"
)

// CodexAdapter installs artifacts into OpenAI Codex's ~/.codex/ hierarchy.
type CodexAdapter struct{}

func NewCodexAdapter() *CodexAdapter { return &CodexAdapter{} }

func (a *CodexAdapter) Name() string { return "codex" }

func (a *CodexAdapter) Detect() bool {
	home, _ := os.UserHomeDir()
	if _, err := os.Stat(filepath.Join(home, ".codex")); err == nil {
		return true
	}
	_, err := exec.LookPath("codex")
	return err == nil
}

func (a *CodexAdapter) InstallPath(t registry.ArtifactType, scope Scope) (string, error) {
	base, err := a.base(scope)
	if err != nil {
		return "", err
	}
	switch t {
	case registry.TypeSkill:
		return filepath.Join(base, "skills"), nil
	case registry.TypeSlashCommand:
		return filepath.Join(base, "prompts"), nil
	case registry.TypeMCPServer:
		return base, nil // written into config.toml
	default:
		return base, nil
	}
}

func (a *CodexAdapter) Install(art *registry.Artifact, payload string, scope Scope) error {
	switch art.Type {
	case registry.TypeSkill:
		return a.installSkill(art, payload, scope)
	case registry.TypeMCPServer:
		return a.installMCP(art, scope)
	case registry.TypeSlashCommand:
		return a.installPrompt(art, payload, scope)
	default:
		return fmt.Errorf("codex: installing %q artifacts is not yet supported", art.Type)
	}
}

func (a *CodexAdapter) installSkill(art *registry.Artifact, payload string, scope Scope) error {
	base, err := a.base(scope)
	if err != nil {
		return err
	}
	// User scope: ~/.codex/skills/; project scope: .agents/skills/
	var skillsDir string
	if scope == ScopeProject {
		cwd, _ := os.Getwd()
		skillsDir = filepath.Join(cwd, ".agents", "skills", shortName(art.Name))
	} else {
		skillsDir = filepath.Join(base, "skills", shortName(art.Name))
	}
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", skillsDir, err)
	}
	return copyDir(payload, skillsDir)
}

// installMCP appends an [mcp_servers.<name>] table to ~/.codex/config.toml.
func (a *CodexAdapter) installMCP(art *registry.Artifact, scope Scope) error {
	cfg, ok := art.Install["codex"]
	if !ok {
		cfg, ok = art.Install["claude-code"]
	}
	if !ok {
		cfg, ok = art.Install["any"]
	}
	if !ok || cfg.MCPServer == nil {
		return fmt.Errorf("artifact %q has no codex MCP install config", art.Name)
	}
	return a.writeMCPToml(art.Name, cfg.MCPServer, scope)
}

func (a *CodexAdapter) writeMCPToml(name string, srv *registry.MCPServerConfig, scope Scope) error {
	cfgPath, err := a.configPath(scope)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o700); err != nil {
		return err
	}

	// Read existing TOML as a raw map so we don't lose unknown fields.
	raw := make(map[string]interface{})
	if data, err := os.ReadFile(cfgPath); err == nil {
		_ = toml.Unmarshal(data, &raw)
	}

	servers, _ := raw["mcp_servers"].(map[string]interface{})
	if servers == nil {
		servers = make(map[string]interface{})
	}
	entry := map[string]interface{}{
		"command": srv.Command,
	}
	if len(srv.Args) > 0 {
		entry["args"] = srv.Args
	}
	if len(srv.Env) > 0 {
		entry["env"] = srv.Env
	}
	if srv.CWD != "" {
		entry["cwd"] = srv.CWD
	}
	servers[shortName(name)] = entry
	raw["mcp_servers"] = servers

	var sb strings.Builder
	enc := toml.NewEncoder(&sb)
	if err := enc.Encode(raw); err != nil {
		return fmt.Errorf("encode TOML: %w", err)
	}
	return writeFileAtomic(cfgPath, []byte(sb.String()), 0o600)
}

func (a *CodexAdapter) installPrompt(art *registry.Artifact, payload string, scope Scope) error {
	base, err := a.base(scope)
	if err != nil {
		return err
	}
	dir := filepath.Join(base, "prompts")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	cfg := art.Install["codex"]
	body := cfg.CommandBody
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

func (a *CodexAdapter) IsInstalled(name string) (bool, error) {
	home, _ := os.UserHomeDir()
	short := shortName(name)
	candidates := []string{
		filepath.Join(home, ".codex", "skills", short),
		filepath.Join(home, ".codex", "prompts", short+".md"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return true, nil
		}
	}
	cfgPath, _ := a.configPath(ScopeUser)
	raw := make(map[string]interface{})
	if data, err := os.ReadFile(cfgPath); err == nil {
		_ = toml.Unmarshal(data, &raw)
		if servers, ok := raw["mcp_servers"].(map[string]interface{}); ok {
			if _, ok := servers[short]; ok {
				return true, nil
			}
		}
	}
	return false, nil
}

func (a *CodexAdapter) Remove(name string, scope Scope) error {
	base, err := a.base(scope)
	if err != nil {
		return err
	}
	short := shortName(name)
	var removed bool
	candidates := []string{
		filepath.Join(base, "skills", short),
		filepath.Join(base, "prompts", short+".md"),
	}
	if scope == ScopeProject {
		cwd, _ := os.Getwd()
		candidates = append(candidates, filepath.Join(cwd, ".agents", "skills", short))
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			if err := os.RemoveAll(p); err != nil {
				return err
			}
			removed = true
		}
	}
	// Remove from config.toml
	cfgPath, _ := a.configPath(scope)
	raw := make(map[string]interface{})
	if data, err := os.ReadFile(cfgPath); err == nil {
		_ = toml.Unmarshal(data, &raw)
		if servers, ok := raw["mcp_servers"].(map[string]interface{}); ok {
			if _, ok := servers[short]; ok {
				delete(servers, short)
				raw["mcp_servers"] = servers
				var sb strings.Builder
				enc := toml.NewEncoder(&sb)
				if err := enc.Encode(raw); err == nil {
					_ = writeFileAtomic(cfgPath, []byte(sb.String()), 0o600)
				}
				removed = true
			}
		}
	}
	if !removed {
		return fmt.Errorf("codex: %q not found (nothing to remove)", name)
	}
	return nil
}

func (a *CodexAdapter) base(scope Scope) (string, error) {
	if scope == ScopeProject {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		return filepath.Join(cwd, ".codex"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".codex"), nil
}

func (a *CodexAdapter) configPath(scope Scope) (string, error) {
	base, err := a.base(scope)
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "config.toml"), nil
}
