# agent-registry

**A free, CLI-first, cross-agent registry for AI agent artifacts.**

`agr` discovers, installs, and manages MCP servers, SKILL.md skills, slash commands, subagents, hooks, and plugin bundles — for both **Claude Code** and **OpenAI Codex** — from a single command.

```
agr search filesystem
agr install io.github.modelcontextprotocol/filesystem
agr install io.github.community/code-review --agent claude-code
```

---

## Why

Every existing registry is single-agent or single-type:
- MCP registries (official, Smithery, Glama) only catalog MCP servers
- `skills.sh` / `npx skills` only handles SKILL.md skills
- Claude Code marketplaces and Codex plugin directories are agent-specific

`agr` unifies **all artifact types** with **cross-agent install adapters**, layered curation/trust, and explicit security disclosure — all from a **free, Git-repo-backed static index**.

---

## Installation

```bash
# From source
go install github.com/rohithilluri/agent-registry@latest

# Homebrew (coming soon)
brew install rohithilluri/tap/agr

# Install script (coming soon)
curl -fsSL https://agr.dev/install.sh | sh
```

---

## Usage

### Search

```bash
agr search <query>                        # full-text search
agr search --type mcp-server              # filter by artifact type
agr search --agent codex                  # filter by agent compatibility
agr search --category databases           # filter by category
agr search git --type skill               # combine filters
```

### Info

```bash
agr info io.github.modelcontextprotocol/filesystem
agr info code-review                      # short name works too
```

### Install

```bash
agr install io.github.modelcontextprotocol/filesystem
agr install io.github.community/code-review --agent claude-code
agr install io.github.community/git-workflow --global  # user scope (default)
agr install io.github.community/test-writer --project  # current project scope
agr install io.github.community/code-review --yes      # skip confirmation
```

`agr install` always shows a **permission disclosure** before writing anything:

```
── Install Disclosure ──────────────────────────────────���────
  Artifact : io.github.modelcontextprotocol/filesystem (mcp-server)
  Version  : 0.6.2
  Trust    : curated
  Perms    : read_files, write_files, list_directories
  ⚠  Contains an MCP server (runs as an external process)
────────────────────────────────────────────────────────────

Install "io.github.modelcontextprotocol/filesystem" for claude-code? [y/N]
```

### List installed artifacts

```bash
agr list
agr list --agent claude-code
```

### Remove

```bash
agr remove io.github.community/code-review
agr remove filesystem --agent codex --yes
```

### Scaffold a new artifact

```bash
agr init skill my-skill-name      # creates my-skill-name/SKILL.md + scripts/, references/
agr init mcp my-mcp-server        # creates my-mcp-server/server.json + .mcp.json + README.md
agr init plugin my-bundle         # creates both .claude-plugin/ and .codex-plugin/ manifests
```

### Publish to the registry

```bash
agr publish ./my-skill            # auto-detects type, validates, prints manifest JSON
agr publish ./my-mcp --type mcp-server --name io.github.me/my-mcp --out manifest.json
```

Then open a PR adding `manifest.json` to `registry/artifacts/` and an IndexEntry to `registry/index.json`. CI validates schema, artifact fields, and runs security scans automatically.

---

## Artifact types

| Type | Claude Code path | Codex path |
|------|-----------------|------------|
| `skill` | `~/.claude/skills/<name>/` | `~/.codex/skills/<name>/` |
| `mcp-server` | `~/.claude/settings.json` or `.mcp.json` | `~/.codex/config.toml` |
| `slash-command` | `~/.claude/commands/<name>.md` | `~/.codex/prompts/<name>.md` |
| `subagent` | `~/.claude/agents/<name>.md` | — |
| `plugin-bundle` | `.claude-plugin/` | `.codex-plugin/` |

---

## Trust model

| Tier | Meaning |
|------|---------|
| `curated` | Hand-reviewed by maintainers, featured in the official registry |
| `verified` | Namespace ownership verified (GitHub/DNS); clean scan history |
| `community` | PR-submitted; automated schema validation + security scan |

Every artifact has a recorded **sha256 checksum** verified at install time. Hooks, MCP servers, and scripted skills show explicit warnings before install.

---

## Adding to the registry

### As a Claude Code marketplace

In Claude Code, run:

```
/plugin marketplace add https://github.com/rohithilluri/agent-registry
```

Then install any skill with `/plugin install code-review`.

### Via PR (full registry entry)

1. Fork this repo
2. Run `agr publish ./your-artifact --name io.github.YOURUSER/your-artifact`
3. Add the manifest to `registry/artifacts/io.github.YOURUSER/your-artifact.json`
4. Add an `IndexEntry` to `registry/index.json`
5. Open a PR — CI validates schema, fields, and runs mcp-scan

#### Namespace rules

- **GitHub namespaces**: `io.github.<username>/<artifact>` — proven by GitHub identity
- **DNS namespaces**: `com.yourco/<artifact>` — proven by DNS TXT record
- Use kebab-case for the artifact segment

---

## Registry architecture

```
registry/
├── index.json               # compact entries for client-side search
└── artifacts/               # per-artifact full manifests (sparse index)
    ├── io.github.modelcontextprotocol/
    │   ├── filesystem.json
    │   └── ...
    └── io.github.community/
        ├── code-review.json
        └── ...

skills/                      # bundled SKILL.md skills (shipped with this repo)
├── code-review/SKILL.md
├── git-workflow/SKILL.md
├── docker-ops/SKILL.md
├── test-writer/SKILL.md
└── sql-helper/SKILL.md

.claude-plugin/
└── marketplace.json         # Claude Code marketplace integration
```

**Hosting cost: $0/month.** The index is served via GitHub raw/Pages. Payloads live on npm/PyPI/GitHub Releases. No always-on server or database.

---

## Adapter architecture

```
internal/
├── registry/     types, index fetch/cache, search
├── adapter/      Adapter interface + ClaudeCodeAdapter + CodexAdapter
├── installer/    download → verify checksum → disclose → adapt → install
├── security/     sha256 checksum verification
└── config/       CLI config, installed DB
```

The `Adapter` interface makes adding more agents (Cursor, Copilot, Gemini CLI, Goose) a matter of implementing six methods. Both existing adapters detect agent presence by probing `~/.claude`/`~/.codex` and the system PATH.

---

## Security

- **Checksums**: every indexed artifact has a `sha256:` checksum verified at install time
- **Disclosure**: install always prints artifact type, permissions, and executable-code warnings
- **Confirmation**: required unless `--yes` is passed
- **Trust tiers**: community artifacts are scanned by CI (mcp-scan) but not human-reviewed
- **Path traversal guard**: tar extraction validates all paths stay within the destination dir
- **No auto-hooks**: hooks require explicit opt-in; never silently enabled

---

## Roadmap

- **Stage 1 (MVP — this PR)**: MCP + skill install for Claude Code + Codex, static index, CI validation
- **Stage 2**: Verified-publisher tier (GitHub/DNS namespace proof), Sigstore/cosign provenance, consume official MCP Registry as upstream, slash-command/subagent/hook types
- **Stage 3**: Adapters for Cursor, Copilot, Gemini CLI, Goose; install-count leaderboard; conform to MCP Registry OpenAPI as a subregistry

---

## Contributing

Contributions welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) (coming soon) for the PR workflow.

License: MIT
