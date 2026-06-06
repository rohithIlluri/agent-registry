package adapter_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/rohithilluri/agent-registry/internal/adapter"
	"github.com/rohithilluri/agent-registry/internal/registry"
)

// setHome redirects HOME to a temp dir so tests don't touch the real ~/.claude / ~/.codex.
func setHome(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	return tmp
}

// --- ClaudeCodeAdapter ---

func TestClaudeCode_Name(t *testing.T) {
	a := adapter.NewClaudeCodeAdapter()
	if a.Name() != "claude-code" {
		t.Errorf("expected 'claude-code', got %q", a.Name())
	}
}

func TestClaudeCode_Detect_ByDir(t *testing.T) {
	home := setHome(t)
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	a := adapter.NewClaudeCodeAdapter()
	if !a.Detect() {
		t.Error("expected Detect to return true when ~/.claude exists")
	}
}

func TestClaudeCode_Detect_False(t *testing.T) {
	setHome(t)
	// No ~/.claude and 'claude' not on PATH in test env.
	a := adapter.NewClaudeCodeAdapter()
	// We can't control PATH easily, but in a clean temp HOME this should be false
	// unless the real claude binary is installed. Just verify it doesn't panic.
	_ = a.Detect()
}

func TestClaudeCode_InstallSkill(t *testing.T) {
	home := setHome(t)

	// Prepare a fake skill payload dir
	skillDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: test\ndescription: x\n---\nbody"), 0o644); err != nil {
		t.Fatal(err)
	}

	art := &registry.Artifact{
		Name:    "io.github.test/my-skill",
		Type:    registry.TypeSkill,
		Version: "1.0.0",
		Install: registry.InstallConfig{},
	}

	a := adapter.NewClaudeCodeAdapter()
	if err := a.Install(art, skillDir, adapter.ScopeUser); err != nil {
		t.Fatalf("Install skill: %v", err)
	}

	dest := filepath.Join(home, ".claude", "skills", "my-skill", "SKILL.md")
	if _, err := os.Stat(dest); err != nil {
		t.Errorf("expected SKILL.md at %s: %v", dest, err)
	}
}

func TestClaudeCode_InstallMCP_ProjectScope(t *testing.T) {
	// Use a temp dir as the "project root".
	projectDir := t.TempDir()
	if err := os.Chdir(projectDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(os.TempDir()) })

	art := &registry.Artifact{
		Name:    "io.github.test/my-mcp",
		Type:    registry.TypeMCPServer,
		Version: "1.0.0",
		Install: registry.InstallConfig{
			"claude-code": {
				MCPServer: &registry.MCPServerConfig{
					Command: "npx",
					Args:    []string{"-y", "my-package"},
				},
			},
		},
	}

	a := adapter.NewClaudeCodeAdapter()
	if err := a.Install(art, "", adapter.ScopeProject); err != nil {
		t.Fatalf("Install MCP: %v", err)
	}

	mcpFile := filepath.Join(projectDir, ".mcp.json")
	data, err := os.ReadFile(mcpFile)
	if err != nil {
		t.Fatalf("expected .mcp.json: %v", err)
	}
	var f map[string]interface{}
	if err := json.Unmarshal(data, &f); err != nil {
		t.Fatalf("parse .mcp.json: %v", err)
	}
	servers, _ := f["mcpServers"].(map[string]interface{})
	if servers == nil {
		t.Fatal("mcpServers not found in .mcp.json")
	}
	if _, ok := servers["my-mcp"]; !ok {
		t.Errorf("expected 'my-mcp' entry in mcpServers, got keys: %v", servers)
	}
}

func TestClaudeCode_InstallMCP_UserScope(t *testing.T) {
	home := setHome(t)

	art := &registry.Artifact{
		Name:    "io.github.test/my-mcp",
		Type:    registry.TypeMCPServer,
		Version: "1.0.0",
		Install: registry.InstallConfig{
			"claude-code": {
				MCPServer: &registry.MCPServerConfig{
					Command: "npx",
					Args:    []string{"-y", "my-pkg"},
				},
			},
		},
	}

	a := adapter.NewClaudeCodeAdapter()
	if err := a.Install(art, "", adapter.ScopeUser); err != nil {
		t.Fatalf("Install MCP user: %v", err)
	}

	settingsPath := filepath.Join(home, ".claude", "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("expected settings.json: %v", err)
	}
	var s map[string]interface{}
	_ = json.Unmarshal(data, &s)
	servers, _ := s["mcpServers"].(map[string]interface{})
	if _, ok := servers["my-mcp"]; !ok {
		t.Errorf("expected 'my-mcp' in settings.json mcpServers, got %v", servers)
	}
}

func TestClaudeCode_IsInstalled_Skill(t *testing.T) {
	home := setHome(t)
	skillPath := filepath.Join(home, ".claude", "skills", "my-skill")
	if err := os.MkdirAll(skillPath, 0o755); err != nil {
		t.Fatal(err)
	}

	a := adapter.NewClaudeCodeAdapter()
	ok, err := a.IsInstalled("io.github.test/my-skill")
	if err != nil {
		t.Fatalf("IsInstalled: %v", err)
	}
	if !ok {
		t.Error("expected IsInstalled to return true")
	}
}

func TestClaudeCode_Remove_Skill(t *testing.T) {
	home := setHome(t)
	skillPath := filepath.Join(home, ".claude", "skills", "my-skill")
	if err := os.MkdirAll(skillPath, 0o755); err != nil {
		t.Fatal(err)
	}

	a := adapter.NewClaudeCodeAdapter()
	if err := a.Remove("io.github.test/my-skill", adapter.ScopeUser); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, err := os.Stat(skillPath); !os.IsNotExist(err) {
		t.Error("expected skill dir to be removed")
	}
}

func TestClaudeCode_InstallHook(t *testing.T) {
	home := setHome(t)

	art := &registry.Artifact{
		Name:    "io.github.test/my-hook",
		Type:    registry.TypeHook,
		Version: "1.0.0",
		Install: registry.InstallConfig{
			"claude-code": {
				Hooks: registry.HookDefinition{
					"PreToolUse": {
						{
							Matcher: "Bash",
							Hooks: []registry.HookCommand{
								{Type: "command", Command: "echo pre"},
							},
						},
					},
				},
			},
		},
	}

	a := adapter.NewClaudeCodeAdapter()
	if err := a.Install(art, "", adapter.ScopeUser); err != nil {
		t.Fatalf("Install hook: %v", err)
	}

	settingsPath := filepath.Join(home, ".claude", "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("expected settings.json: %v", err)
	}
	var s map[string]interface{}
	_ = json.Unmarshal(data, &s)
	hooks, _ := s["hooks"].(map[string]interface{})
	if hooks == nil {
		t.Fatal("hooks missing from settings.json")
	}
	preToolUse, _ := hooks["PreToolUse"].([]interface{})
	if len(preToolUse) == 0 {
		t.Error("PreToolUse hooks should not be empty")
	}
}

func TestClaudeCode_InstallPlugin(t *testing.T) {
	home := setHome(t)

	pluginPayload := t.TempDir()
	if err := os.WriteFile(filepath.Join(pluginPayload, "plugin.json"), []byte(`{"name":"test-plugin"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	art := &registry.Artifact{
		Name:    "io.github.test/my-plugin",
		Type:    registry.TypePlugin,
		Version: "1.0.0",
		Install: registry.InstallConfig{},
	}

	a := adapter.NewClaudeCodeAdapter()
	if err := a.Install(art, pluginPayload, adapter.ScopeUser); err != nil {
		t.Fatalf("Install plugin: %v", err)
	}

	// Cache dir should exist
	cacheDir := filepath.Join(home, ".claude", "plugins", "cache", "my-plugin")
	if _, err := os.Stat(cacheDir); err != nil {
		t.Errorf("expected plugin cache at %s: %v", cacheDir, err)
	}

	// Plugin should be in enabledPlugins
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("expected settings.json: %v", err)
	}
	var s map[string]interface{}
	_ = json.Unmarshal(data, &s)
	enabled, _ := s["enabledPlugins"].([]interface{})
	if len(enabled) == 0 {
		t.Fatal("enabledPlugins should not be empty")
	}
	if enabled[0] != "my-plugin" {
		t.Errorf("enabledPlugins[0] = %v, want my-plugin", enabled[0])
	}
}

func TestClaudeCode_Remove_Plugin(t *testing.T) {
	home := setHome(t)

	// Seed a plugin
	cacheDir := filepath.Join(home, ".claude", "plugins", "cache", "my-plugin")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	settings := map[string]interface{}{
		"enabledPlugins": []string{"my-plugin"},
	}
	data, _ := json.MarshalIndent(settings, "", "  ")
	_ = os.MkdirAll(filepath.Join(home, ".claude"), 0o755)
	_ = os.WriteFile(filepath.Join(home, ".claude", "settings.json"), data, 0o600)

	a := adapter.NewClaudeCodeAdapter()
	if err := a.Remove("io.github.test/my-plugin", adapter.ScopeUser); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	// Cache dir should be gone
	if _, err := os.Stat(cacheDir); !os.IsNotExist(err) {
		t.Error("expected plugin cache to be removed")
	}
}

// --- CodexAdapter ---

func TestCodex_Name(t *testing.T) {
	a := adapter.NewCodexAdapter()
	if a.Name() != "codex" {
		t.Errorf("expected 'codex', got %q", a.Name())
	}
}

func TestCodex_InstallSkill(t *testing.T) {
	home := setHome(t)

	skillDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: test\ndescription: x\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	art := &registry.Artifact{
		Name:    "io.github.test/my-skill",
		Type:    registry.TypeSkill,
		Version: "1.0.0",
		Install: registry.InstallConfig{},
	}

	a := adapter.NewCodexAdapter()
	if err := a.Install(art, skillDir, adapter.ScopeUser); err != nil {
		t.Fatalf("Install skill: %v", err)
	}

	dest := filepath.Join(home, ".codex", "skills", "my-skill", "SKILL.md")
	if _, err := os.Stat(dest); err != nil {
		t.Errorf("expected SKILL.md at %s: %v", dest, err)
	}
}

func TestCodex_InstallMCP(t *testing.T) {
	home := setHome(t)

	art := &registry.Artifact{
		Name:    "io.github.test/my-mcp",
		Type:    registry.TypeMCPServer,
		Version: "1.0.0",
		Install: registry.InstallConfig{
			"codex": {
				MCPServer: &registry.MCPServerConfig{
					Command: "npx",
					Args:    []string{"-y", "my-pkg"},
				},
			},
		},
	}

	a := adapter.NewCodexAdapter()
	if err := a.Install(art, "", adapter.ScopeUser); err != nil {
		t.Fatalf("Install MCP: %v", err)
	}

	cfgPath := filepath.Join(home, ".codex", "config.toml")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("expected config.toml: %v", err)
	}
	content := string(data)
	if content == "" {
		t.Error("config.toml should not be empty")
	}
	// The TOML must contain the server name
	if !containsSubstr(content, "my-mcp") {
		t.Errorf("expected 'my-mcp' in config.toml, got:\n%s", content)
	}
}

// --- ByName / All / Detect ---

func TestByName(t *testing.T) {
	a := adapter.ByName("claude-code")
	if a == nil || a.Name() != "claude-code" {
		t.Errorf("ByName('claude-code') returned wrong adapter: %v", a)
	}
	b := adapter.ByName("codex")
	if b == nil || b.Name() != "codex" {
		t.Errorf("ByName('codex') returned wrong adapter: %v", b)
	}
	if adapter.ByName("nonexistent") != nil {
		t.Error("ByName('nonexistent') should return nil")
	}
}

func TestAll(t *testing.T) {
	all := adapter.All()
	if len(all) < 2 {
		t.Errorf("expected at least 2 adapters, got %d", len(all))
	}
	names := map[string]bool{}
	for _, a := range all {
		names[a.Name()] = true
	}
	for _, want := range []string{"claude-code", "codex"} {
		if !names[want] {
			t.Errorf("missing adapter %q in All()", want)
		}
	}
}

func containsSubstr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && func() bool {
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	}())
}
