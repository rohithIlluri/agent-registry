package adapter

import "github.com/rohithilluri/agent-registry/internal/registry"

// Scope controls whether the artifact installs for the current project or the user globally.
type Scope string

const (
	ScopeUser    Scope = "user"
	ScopeProject Scope = "project"
)

// Adapter is implemented by each supported agent.
type Adapter interface {
	// Name returns the canonical agent identifier (e.g. "claude-code", "codex").
	Name() string

	// Detect returns true if this agent appears to be installed on the system.
	Detect() bool

	// Install writes the artifact into the agent's config / skill directory.
	// payload is the local filesystem path to the downloaded artifact (may be a dir or archive).
	Install(art *registry.Artifact, payload string, scope Scope) error

	// IsInstalled reports whether an artifact by name is already installed.
	IsInstalled(name string) (bool, error)

	// Remove uninstalls a previously installed artifact.
	Remove(name string, scope Scope) error

	// InstallPath returns the directory where the artifact would be installed.
	InstallPath(artType registry.ArtifactType, scope Scope) (string, error)

	// SupportedTypes lists the artifact types this agent can install.
	SupportedTypes() []registry.ArtifactType
}

// All returns every adapter, in preference order.
func All() []Adapter {
	return []Adapter{
		NewClaudeCodeAdapter(),
		NewCodexAdapter(),
	}
}

// Detect returns the subset of adapters whose agents are detected on the system.
func Detect() []Adapter {
	var found []Adapter
	for _, a := range All() {
		if a.Detect() {
			found = append(found, a)
		}
	}
	return found
}

// ByName returns the named adapter, or nil.
func ByName(name string) Adapter {
	for _, a := range All() {
		if a.Name() == name {
			return a
		}
	}
	return nil
}
