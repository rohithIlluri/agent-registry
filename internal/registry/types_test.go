package registry

import "testing"

func TestInstallFor_AgentKeyWins(t *testing.T) {
	art := &Artifact{Install: InstallConfig{
		"claude-code": {CommandBody: "claude"},
		"any":         {CommandBody: "fallback"},
	}}
	cfg, ok := art.InstallFor("claude-code")
	if !ok || cfg.CommandBody != "claude" {
		t.Errorf("InstallFor(claude-code) = %q, %v; want claude, true", cfg.CommandBody, ok)
	}
}

func TestInstallFor_FallsBackToAny(t *testing.T) {
	art := &Artifact{Install: InstallConfig{
		"any": {CommandBody: "fallback"},
	}}
	cfg, ok := art.InstallFor("codex")
	if !ok || cfg.CommandBody != "fallback" {
		t.Errorf("InstallFor(codex) = %q, %v; want fallback, true", cfg.CommandBody, ok)
	}
}

func TestInstallFor_TriesKeysInOrder(t *testing.T) {
	art := &Artifact{Install: InstallConfig{
		"claude-code": {CommandBody: "claude"},
	}}
	cfg, ok := art.InstallFor("codex", "claude-code")
	if !ok || cfg.CommandBody != "claude" {
		t.Errorf("InstallFor(codex, claude-code) = %q, %v; want claude, true", cfg.CommandBody, ok)
	}
}

func TestInstallFor_NoMatch(t *testing.T) {
	art := &Artifact{Install: InstallConfig{}}
	if _, ok := art.InstallFor("codex"); ok {
		t.Error("InstallFor on empty config should return ok=false")
	}
}
