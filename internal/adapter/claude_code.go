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

func (a *ClaudeCodeAdapter) SupportedTypes() []registry.ArtifactType {
	return []registry.ArtifactType{
		registry.TypeSkill,
		registry.TypeMCPServer,
		registry.TypeSlashCommand,
		registry.TypeSubagent,
		registry.TypeHook,
		registry.TypePlugin,
	}
}

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
	case registry.TypeHook:
		return a.installHook(art, scope)
	case registry.TypePlugin:
		return a.installPlugin(art, payload, scope)
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
	if err := os.MkdirAll(dest, 0o750); err != nil {
		return fmt.Errorf("mkdir %s: %w", dest, err)
	}
	return copyDir(payload, dest)
}

// installMCP writes a server entry into .mcp.json (project) or ~/.claude/settings.json (user).
func (a *ClaudeCodeAdapter) installMCP(art *registry.Artifact, scope Scope) error {
	cfg, ok := art.InstallFor(a.Name())
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
	if data, err := os.ReadFile(path); err == nil { // #nosec G304 -- path is project CWD .mcp.json
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

type claudeHookMatcher struct {
	Matcher string                 `json:"matcher,omitempty"`
	Hooks   []registry.HookCommand `json:"hooks"`
}

type claudeSettings struct {
	MCPServers     map[string]registry.MCPServerConfig `json:"mcpServers,omitempty"`
	Hooks          map[string][]claudeHookMatcher      `json:"hooks,omitempty"`
	EnabledPlugins []string                            `json:"enabledPlugins,omitempty"`
}

func (a *ClaudeCodeAdapter) writeSettingsMCP(name string, srv *registry.MCPServerConfig) error {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, ".claude", "settings.json")
	var s claudeSettings
	if data, err := os.ReadFile(path); err == nil { // #nosec G304 -- path is ~/.claude/settings.json
		_ = json.Unmarshal(data, &s)
	}
	if s.MCPServers == nil {
		s.MCPServers = make(map[string]registry.MCPServerConfig)
	}
	s.MCPServers[shortName(name)] = *srv
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
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
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	// payload is the .md file, or we use the inline body
	cfg, _ := art.InstallFor(a.Name())
	body := cfg.CommandBody
	if body == "" && payload != "" {
		data, err := os.ReadFile(payload) // #nosec G304 -- payload path from installer, not user input
		if err != nil {
			return err
		}
		body = string(data)
	}
	if body == "" {
		return fmt.Errorf("artifact %q has no command body or payload to install", art.Name)
	}
	dest := filepath.Join(dir, shortName(art.Name)+".md")
	return os.WriteFile(dest, []byte(body), 0o644) // #nosec G306 -- command files are read by the agent at runtime
}

func (a *ClaudeCodeAdapter) installSubagent(art *registry.Artifact, payload string, scope Scope) error {
	base, err := a.base(scope)
	if err != nil {
		return err
	}
	dir := filepath.Join(base, "agents")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	cfg, _ := art.InstallFor(a.Name())
	body := cfg.AgentBody
	if body == "" && payload != "" {
		data, err := os.ReadFile(payload) // #nosec G304 -- payload path from installer, not user input
		if err != nil {
			return err
		}
		body = string(data)
	}
	if body == "" {
		return fmt.Errorf("artifact %q has no agent body or payload to install", art.Name)
	}
	dest := filepath.Join(dir, shortName(art.Name)+".md")
	return os.WriteFile(dest, []byte(body), 0o644) // #nosec G306 -- subagent files are read by the agent at runtime
}

// installHook merges hook definitions from the artifact into ~/.claude/settings.json.
func (a *ClaudeCodeAdapter) installHook(art *registry.Artifact, scope Scope) error {
	cfg, ok := art.InstallFor(a.Name())
	if !ok || len(cfg.Hooks) == 0 {
		return fmt.Errorf("artifact %q has no claude-code hook install config", art.Name)
	}
	base, err := a.base(scope)
	if err != nil {
		return err
	}
	path := filepath.Join(base, "settings.json")
	var s claudeSettings
	if data, err := os.ReadFile(path); err == nil { // #nosec G304 -- path is under ~/.claude, not user-supplied input
		_ = json.Unmarshal(data, &s)
	}
	if s.Hooks == nil {
		s.Hooks = make(map[string][]claudeHookMatcher)
	}
	for event, matchers := range cfg.Hooks {
		for _, m := range matchers {
			s.Hooks[event] = append(s.Hooks[event], claudeHookMatcher{
				Matcher: m.Matcher,
				Hooks:   m.Hooks,
			})
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}
	return writeFileAtomic(path, data, 0o600)
}

// installPlugin downloads the bundle into ~/.claude/plugins/cache/<name>/ and
// registers the plugin name in settings.json enabledPlugins.
func (a *ClaudeCodeAdapter) installPlugin(art *registry.Artifact, payload string, scope Scope) error {
	base, err := a.base(scope)
	if err != nil {
		return err
	}
	cacheDir := filepath.Join(base, "plugins", "cache", shortName(art.Name))
	if err := os.MkdirAll(cacheDir, 0o750); err != nil {
		return fmt.Errorf("mkdir %s: %w", cacheDir, err)
	}
	if payload != "" {
		if err := copyDir(payload, cacheDir); err != nil {
			return err
		}
	}
	// Register in settings.json enabledPlugins.
	path := filepath.Join(base, "settings.json")
	var s claudeSettings
	if data, err := os.ReadFile(path); err == nil { // #nosec G304 -- path is under ~/.claude, not user-supplied input
		_ = json.Unmarshal(data, &s)
	}
	short := shortName(art.Name)
	for _, p := range s.EnabledPlugins {
		if p == short {
			return nil // already registered
		}
	}
	s.EnabledPlugins = append(s.EnabledPlugins, short)
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}
	return writeFileAtomic(path, data, 0o600)
}

func (a *ClaudeCodeAdapter) IsInstalled(name string) (bool, error) {
	home, _ := os.UserHomeDir()
	short := shortName(name)
	candidates := []string{
		filepath.Join(home, ".claude", "skills", short),
		filepath.Join(home, ".claude", "commands", short+".md"),
		filepath.Join(home, ".claude", "agents", short+".md"),
		filepath.Join(home, ".claude", "plugins", "cache", short),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return true, nil
		}
	}
	// Check MCP / enabledPlugins in settings.json
	path := filepath.Join(home, ".claude", "settings.json")
	var s claudeSettings
	if data, err := os.ReadFile(path); err == nil { // #nosec G304 -- path is ~/.claude/settings.json
		_ = json.Unmarshal(data, &s)
		if _, ok := s.MCPServers[short]; ok {
			return true, nil
		}
		for _, p := range s.EnabledPlugins {
			if p == short {
				return true, nil
			}
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
		filepath.Join(base, "plugins", "cache", short),
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
	// Remove from settings.json (MCP + enabledPlugins)
	settingsPath := filepath.Join(base, "settings.json")
	var s claudeSettings
	if data, err := os.ReadFile(settingsPath); err == nil { // #nosec G304 -- path is under ~/.claude, not user-supplied input
		_ = json.Unmarshal(data, &s)
		changed := false
		if _, ok := s.MCPServers[short]; ok {
			delete(s.MCPServers, short)
			changed = true
		}
		filtered := s.EnabledPlugins[:0]
		for _, p := range s.EnabledPlugins {
			if p == short {
				changed = true
			} else {
				filtered = append(filtered, p)
			}
		}
		s.EnabledPlugins = filtered
		if changed {
			if data, err := json.MarshalIndent(s, "", "  "); err == nil {
				_ = writeFileAtomic(settingsPath, data, 0o600)
			}
			removed = true
		}
	}
	// Remove from project .mcp.json
	if scope == ScopeProject {
		cwd, _ := os.Getwd()
		path := filepath.Join(cwd, ".mcp.json")
		var f mcpJSONFile
		if data, err := os.ReadFile(path); err == nil { // #nosec G304 -- path is project CWD .mcp.json
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
