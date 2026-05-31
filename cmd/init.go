package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init <type> <name>",
		Short: "Scaffold a new artifact",
		Long:  "Scaffold skeleton files for a new skill, MCP server, or plugin bundle.",
		Args:  cobra.ExactArgs(2),
		Example: `  agr init skill my-code-review
  agr init mcp my-db-server
  agr init plugin my-bundle`,
		RunE: func(cmd *cobra.Command, args []string) error {
			artType := strings.ToLower(args[0])
			name := args[1]
			switch artType {
			case "skill":
				return initSkill(name)
			case "mcp":
				return initMCP(name)
			case "plugin":
				return initPlugin(name)
			default:
				return fmt.Errorf("unknown type %q; choose: skill, mcp, plugin", artType)
			}
		},
	}
	return cmd
}

func initSkill(name string) error {
	dir := filepath.Join(".", name)
	if err := os.MkdirAll(filepath.Join(dir, "scripts"), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(dir, "references"), 0o755); err != nil {
		return err
	}

	skillMD := fmt.Sprintf(`---
name: %s
description: "TODO: describe what this skill does (required)"
version: "1.0.0"
license: MIT
keywords: []
category: productivity
# allowed-tools: [Bash, Read, Write, Edit]
# user-invocable: true
---

# %s

TODO: Describe this skill. What does it do? When should the agent invoke it?

## Usage

Describe how to use this skill and what it expects.

## Examples

Provide concrete examples of when this skill fires.
`, name, name)

	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(skillMD), 0o644); err != nil {
		return err
	}

	fmt.Printf("✓ Scaffolded skill: %s/\n", dir)
	fmt.Printf("  Edit %s/SKILL.md to fill in name and description.\n", dir)
	fmt.Printf("  Then run: agr publish ./%s\n", name)
	return nil
}

func initMCP(name string) error {
	dir := filepath.Join(".", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	serverJSON := fmt.Sprintf(`{
  "$schema": "https://registry.modelcontextprotocol.io/schema/2025-12-11/server.json",
  "name": "io.github.YOURUSER/%s",
  "description": "TODO: describe what this MCP server provides (max 100 chars)",
  "version": "1.0.0",
  "repository": "https://github.com/YOURUSER/%s",
  "packages": [
    {
      "registryType": "npm",
      "registryBaseUrl": "https://registry.npmjs.org",
      "identifier": "YOURPACKAGE",
      "version": "1.0.0",
      "transport": "stdio"
    }
  ]
}
`, name, name)

	if err := os.WriteFile(filepath.Join(dir, "server.json"), []byte(serverJSON), 0o644); err != nil {
		return err
	}

	mcpJSON := fmt.Sprintf(`{
  "mcpServers": {
    "%s": {
      "command": "npx",
      "args": ["-y", "YOURPACKAGE"]
    }
  }
}
`, name)

	if err := os.WriteFile(filepath.Join(dir, ".mcp.json"), []byte(mcpJSON), 0o644); err != nil {
		return err
	}

	readmeMD := fmt.Sprintf(`# %s MCP Server

TODO: Describe your MCP server.

## Installation

`+"```"+`
agr install io.github.YOURUSER/%s
`+"```"+`

## Tools

| Tool | Description |
|------|-------------|
| TODO | TODO        |

## Configuration

| Variable | Required | Description |
|----------|----------|-------------|
| TODO     | No       | TODO        |
`, name, name)

	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(readmeMD), 0o644); err != nil {
		return err
	}

	fmt.Printf("✓ Scaffolded MCP server: %s/\n", dir)
	fmt.Printf("  Fill in %s/server.json and implement your server.\n", dir)
	fmt.Printf("  Then run: agr publish ./%s --type mcp-server\n", name)
	return nil
}

// titleCase uppercases the first letter of each space-separated word.
// Avoids the deprecated strings.Title.
func titleCase(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) == 0 {
			continue
		}
		runes := []rune(w)
		runes[0] = unicode.ToUpper(runes[0])
		words[i] = string(runes)
	}
	return strings.Join(words, " ")
}

func initPlugin(name string) error {
	claudeDir := filepath.Join(".", name, ".claude-plugin")
	codexDir := filepath.Join(".", name, ".codex-plugin")
	for _, d := range []string{claudeDir, codexDir, filepath.Join(".", name, "skills"), filepath.Join(".", name, "commands")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}

	claudePlugin := fmt.Sprintf(`{
  "name": "%s",
  "displayName": "%s",
  "version": "1.0.0",
  "description": "TODO: describe this plugin bundle",
  "author": {"name": "YOURNAME", "url": "https://github.com/YOURUSER"},
  "license": "MIT",
  "keywords": [],
  "skills": "skills/",
  "commands": "commands/"
}
`, name, titleCase(strings.ReplaceAll(name, "-", " ")))

	codexPlugin := fmt.Sprintf(`{
  "name": "%s",
  "version": "1.0.0",
  "description": "TODO: describe this plugin bundle",
  "skills": "skills/",
  "interface": {
    "displayName": "%s",
    "category": "productivity"
  }
}
`, name, titleCase(strings.ReplaceAll(name, "-", " ")))

	if err := os.WriteFile(filepath.Join(claudeDir, "plugin.json"), []byte(claudePlugin), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(codexDir, "plugin.json"), []byte(codexPlugin), 0o644); err != nil {
		return err
	}

	fmt.Printf("✓ Scaffolded plugin bundle: %s/\n", name)
	fmt.Printf("  .claude-plugin/plugin.json  → Claude Code manifest\n")
	fmt.Printf("  .codex-plugin/plugin.json   → Codex manifest\n")
	fmt.Printf("  Add skills to %s/skills/ and commands to %s/commands/\n", name, name)
	return nil
}
