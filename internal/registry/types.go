package registry

// ArtifactType enumerates all artifact kinds the registry catalogs.
type ArtifactType string

const (
	TypeMCPServer    ArtifactType = "mcp-server"
	TypeSkill        ArtifactType = "skill"
	TypeSubagent     ArtifactType = "subagent"
	TypeSlashCommand ArtifactType = "slash-command"
	TypeHook         ArtifactType = "hook"
	TypePlugin       ArtifactType = "plugin-bundle"
)

// TrustTier reflects the curation level of an artifact.
type TrustTier string

const (
	TrustCommunity TrustTier = "community"
	TrustVerified  TrustTier = "verified"
	TrustCurated   TrustTier = "curated"
)

// SourceType represents where the artifact payload lives.
type SourceType string

const (
	SourceNPM           SourceType = "npm"
	SourcePyPI          SourceType = "pypi"
	SourceGitHubRelease SourceType = "github-release"
	SourceGit           SourceType = "git"
	SourceLocal         SourceType = "local"
)

type Author struct {
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
	URL   string `json:"url,omitempty"`
}

type Source struct {
	Type    SourceType `json:"type"`
	Package string     `json:"package,omitempty"` // npm package name or pypi name
	Repo    string     `json:"repo,omitempty"`    // github owner/repo
	URL     string     `json:"url,omitempty"`     // direct download URL
	Version string     `json:"version,omitempty"`
}

// MCPServerConfig is the transport-level config for an MCP server.
type MCPServerConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	CWD     string            `json:"cwd,omitempty"`
	URL     string            `json:"url,omitempty"` // for streamable-HTTP transport
}

// HookCommand is a single hook action entry (always type "command" for now).
type HookCommand struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

// HookMatcher pairs an optional tool-name matcher with its hook commands.
type HookMatcher struct {
	Matcher string        `json:"matcher,omitempty"`
	Hooks   []HookCommand `json:"hooks"`
}

// HookDefinition maps Claude Code hook events (PreToolUse, PostToolUse,
// Notification, Stop) to their matchers. Used inside AgentInstallConfig.
type HookDefinition map[string][]HookMatcher

// AgentInstallConfig describes how to install an artifact for a specific agent.
type AgentInstallConfig struct {
	// MCP servers
	MCPServer *MCPServerConfig `json:"mcpServer,omitempty"`

	// Skills / slash-commands / subagents: download source and inner path
	SkillSource string `json:"skillSource,omitempty"` // URL or npm/git ref
	SkillPath   string `json:"skillPath,omitempty"`   // subdirectory inside the archive

	// Inline body for slash-commands or subagents
	CommandBody string `json:"commandBody,omitempty"`
	AgentBody   string `json:"agentBody,omitempty"`

	// Hooks to register (Claude Code only for now)
	Hooks HookDefinition `json:"hooks,omitempty"`

	// Plugin bundle: path inside downloaded archive to the agent's manifest dir
	PluginManifestDir string `json:"pluginManifestDir,omitempty"`
}

// InstallConfig maps agent name → install spec; the key "any" means all agents.
type InstallConfig map[string]AgentInstallConfig

// InstallFor resolves the install config for an agent, trying each given key
// in order and finally falling back to "any". The second return is false when
// no key matched.
func (a *Artifact) InstallFor(agents ...string) (AgentInstallConfig, bool) {
	for _, name := range agents {
		if cfg, ok := a.Install[name]; ok {
			return cfg, true
		}
	}
	cfg, ok := a.Install["any"]
	return cfg, ok
}

// Artifact is the full manifest stored per-artifact in registry/artifacts/.
type Artifact struct {
	Name        string       `json:"name"`
	DisplayName string       `json:"displayName,omitempty"`
	Type        ArtifactType `json:"type"`
	Version     string       `json:"version"`
	Description string       `json:"description"`
	Author      Author       `json:"author,omitempty"`
	License     string       `json:"license,omitempty"`
	Homepage    string       `json:"homepage,omitempty"`
	Repository  string       `json:"repository,omitempty"`
	Keywords    []string     `json:"keywords,omitempty"`
	Category    string       `json:"category"`
	AgentCompat []string     `json:"agentCompat"`
	Trust       TrustTier    `json:"trust"`
	Source      Source       `json:"source"`
	// Checksum is sha256:<hex> of the primary downloadable payload.
	Checksum    string        `json:"checksum,omitempty"`
	Permissions []string      `json:"permissions,omitempty"`
	HasScripts  bool          `json:"hasScripts,omitempty"`
	HasMCP      bool          `json:"hasMCP,omitempty"`
	HasHooks    bool          `json:"hasHooks,omitempty"`
	Install     InstallConfig `json:"install,omitempty"`
}

// IndexEntry is the compact form stored in registry/index.json for search.
type IndexEntry struct {
	Name        string       `json:"name"`
	DisplayName string       `json:"displayName,omitempty"`
	Type        ArtifactType `json:"type"`
	Version     string       `json:"version"`
	Description string       `json:"description"`
	Category    string       `json:"category"`
	Keywords    []string     `json:"keywords,omitempty"`
	AgentCompat []string     `json:"agentCompat"`
	Trust       TrustTier    `json:"trust"`
	HasScripts  bool         `json:"hasScripts,omitempty"`
	HasMCP      bool         `json:"hasMCP,omitempty"`
	HasHooks    bool         `json:"hasHooks,omitempty"`
}

// Index is the top-level registry/index.json structure.
type Index struct {
	Version   string       `json:"version"`
	Generated string       `json:"generated"`
	Artifacts []IndexEntry `json:"artifacts"`
}

// InstalledEntry records a locally installed artifact.
type InstalledEntry struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Version     string `json:"version"`
	Agent       string `json:"agent"`
	Scope       string `json:"scope"` // "user" | "project"
	Path        string `json:"path"`
	InstalledAt string `json:"installedAt"`
}

// InstalledDB is the schema of ~/.agent-registry/installed.json.
type InstalledDB struct {
	Entries []InstalledEntry `json:"entries"`
}
