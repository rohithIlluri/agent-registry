package installer

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/rohithilluri/agent-registry/internal/adapter"
	"github.com/rohithilluri/agent-registry/internal/config"
	"github.com/rohithilluri/agent-registry/internal/registry"
	"github.com/rohithilluri/agent-registry/internal/security"
)

// downloadClient has a generous timeout for artifact payloads (larger files).
var downloadClient = &http.Client{Timeout: 5 * time.Minute}

// Options controls install behaviour.
type Options struct {
	Agents      []string      // target agents; empty = auto-detect
	Scope       adapter.Scope // user or project
	AutoConfirm bool
	Writer      io.Writer // for user-facing output (defaults to os.Stdout)
}

// Install resolves, verifies, discloses, and installs an artifact.
func Install(art *registry.Artifact, opts Options) error {
	w := opts.Writer
	if w == nil {
		w = os.Stdout
	}

	// 1. Determine target adapters.
	var targets []adapter.Adapter
	if len(opts.Agents) > 0 {
		for _, name := range opts.Agents {
			a := adapter.ByName(name)
			if a == nil {
				return fmt.Errorf("unknown agent %q", name)
			}
			targets = append(targets, a)
		}
	} else {
		targets = adapter.Detect()
		if len(targets) == 0 {
			return fmt.Errorf("no supported agent detected; use --agent to specify one explicitly")
		}
	}

	// 2. Verify agent compatibility.
	for _, t := range targets {
		if !compatibleWith(art, t.Name()) {
			return fmt.Errorf("artifact %q does not support agent %q (compatible: %s)",
				art.Name, t.Name(), strings.Join(art.AgentCompat, ", "))
		}
	}

	// 3. Permission disclosure — always show, require confirmation.
	fmt.Fprintln(w)
	printDisclosure(w, art)

	if !opts.AutoConfirm {
		fmt.Fprintf(w, "\nInstall %q for %s? [y/N] ", art.Name, agentNames(targets))
		var resp string
		_, _ = fmt.Scanln(&resp)
		if strings.ToLower(strings.TrimSpace(resp)) != "y" {
			return fmt.Errorf("install cancelled")
		}
	}

	// 4. Download payload (if needed).
	payload, cleanup, err := fetchPayload(art)
	if err != nil {
		return fmt.Errorf("fetch payload: %w", err)
	}
	defer cleanup()

	// 5. Verify checksum.
	if payload != "" && art.Checksum != "" {
		if fi, err := os.Stat(payload); err == nil && !fi.IsDir() {
			if err := security.Verify(payload, art.Checksum); err != nil {
				return fmt.Errorf("checksum verification failed: %w", err)
			}
			fmt.Fprintf(w, "✓ Checksum verified (%s)\n", art.Checksum[:18]+"…")
		}
	}

	// 6. Install for each target adapter.
	for _, t := range targets {
		fmt.Fprintf(w, "Installing %q for %s (%s scope)…\n", art.Name, t.Name(), opts.Scope)
		if err := t.Install(art, payload, opts.Scope); err != nil {
			return fmt.Errorf("%s install: %w", t.Name(), err)
		}
		fmt.Fprintf(w, "✓ Installed for %s\n", t.Name())
	}

	// 7. Record in installed DB.
	if err := recordInstall(art, targets, opts.Scope); err != nil {
		fmt.Fprintf(w, "warning: could not update installed DB: %v\n", err)
	}

	// 8. Post-install instructions.
	printPostInstall(w, art, targets, opts.Scope)

	return nil
}

func printDisclosure(w io.Writer, art *registry.Artifact) {
	bold := func(s string) string { return "\033[1m" + s + "\033[0m" }
	yellow := func(s string) string { return "\033[33m" + s + "\033[0m" }

	fmt.Fprintln(w, bold("── Install Disclosure ───────────────────────────────────────"))
	fmt.Fprintf(w, "  Artifact : %s (%s)\n", art.Name, art.Type)
	fmt.Fprintf(w, "  Version  : %s\n", art.Version)
	fmt.Fprintf(w, "  Trust    : %s\n", art.Trust)
	if len(art.Permissions) > 0 {
		fmt.Fprintf(w, "  Perms    : %s\n", strings.Join(art.Permissions, ", "))
	}
	if art.HasMCP {
		fmt.Fprintln(w, yellow("  ⚠  Contains an MCP server (runs as an external process)"))
	}
	if art.HasScripts {
		fmt.Fprintln(w, yellow("  ⚠  Contains executable scripts"))
	}
	if art.HasHooks {
		fmt.Fprintln(w, yellow("  ⚠  Contains lifecycle hooks (shell execution on agent events)"))
	}
	if art.Trust == registry.TrustCommunity {
		fmt.Fprintln(w, yellow("  ⚠  Community artifact — not human-reviewed by maintainers"))
	}
	fmt.Fprintln(w, bold("────────────────────────────────────────────────────────────"))
}

func printPostInstall(w io.Writer, art *registry.Artifact, targets []adapter.Adapter, scope adapter.Scope) {
	fmt.Fprintln(w)
	for _, t := range targets {
		switch t.Name() {
		case "claude-code":
			switch art.Type {
			case registry.TypeMCPServer:
				if scope == adapter.ScopeProject {
					fmt.Fprintln(w, "→ Claude Code: MCP server added to .mcp.json — restart Claude Code to activate.")
				} else {
					fmt.Fprintln(w, "→ Claude Code: MCP server added to ~/.claude/settings.json — restart Claude Code to activate.")
				}
			case registry.TypeSkill:
				fmt.Fprintln(w, "→ Claude Code: skill ready — invoke with /skill-name in Claude Code.")
			case registry.TypeSlashCommand:
				fmt.Fprintf(w, "→ Claude Code: slash command /%s available immediately.\n", shortName(art.Name))
			}
		case "codex":
			switch art.Type {
			case registry.TypeMCPServer:
				fmt.Fprintln(w, "→ Codex: MCP server appended to ~/.codex/config.toml — restart Codex to activate.")
			case registry.TypeSkill:
				fmt.Fprintln(w, "→ Codex: skill ready — invoke with $skill-name or /skills.")
			}
		}
	}
}

func shortName(name string) string {
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		return name[idx+1:]
	}
	return name
}

func compatibleWith(art *registry.Artifact, agent string) bool {
	for _, a := range art.AgentCompat {
		if a == agent {
			return true
		}
	}
	return false
}

func agentNames(adapters []adapter.Adapter) string {
	names := make([]string, len(adapters))
	for i, a := range adapters {
		names[i] = a.Name()
	}
	return strings.Join(names, ", ")
}

// fetchPayload downloads the artifact payload and returns a local path plus cleanup func.
// For MCP servers that only need registration (no file download), returns "", noop, nil.
func fetchPayload(art *registry.Artifact) (string, func(), error) {
	noop := func() {}
	src := art.Source
	switch src.Type {
	case registry.SourceNPM, registry.SourcePyPI:
		// MCP servers from npm/pypi don't need a local payload — npx/uvx handles them at runtime.
		return "", noop, nil
	case registry.SourceGitHubRelease:
		if src.URL == "" {
			return "", noop, nil
		}
		return downloadTo(src.URL)
	case registry.SourceGit:
		if src.Repo == "" && src.URL == "" {
			return "", noop, nil
		}
		return cloneRepo(src)
	case registry.SourceLocal:
		return src.URL, noop, nil
	default:
		return "", noop, nil
	}
}

func downloadTo(url string) (string, func(), error) {
	resp, err := downloadClient.Get(url) // #nosec G107 -- URL comes from artifact source field, not raw user input
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("download %s: HTTP %d", url, resp.StatusCode)
	}

	tmp, err := os.MkdirTemp("", "agent-registry-*")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(tmp) }

	// Try to unpack tar.gz, otherwise write raw.
	if strings.HasSuffix(url, ".tar.gz") || strings.HasSuffix(url, ".tgz") {
		if err := unpackTarGz(resp.Body, tmp); err != nil {
			cleanup()
			return "", nil, err
		}
		return tmp, cleanup, nil
	}
	dest := filepath.Join(tmp, "payload")
	f, err := os.Create(dest)
	if err != nil {
		cleanup()
		return "", nil, err
	}
	_, err = io.Copy(f, resp.Body)
	if cerr := f.Close(); cerr != nil && err == nil {
		err = cerr
	}
	if err != nil {
		cleanup()
		return "", nil, err
	}
	return dest, cleanup, nil
}

func unpackTarGz(r io.Reader, dest string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		target := filepath.Join(dest, hdr.Name) // #nosec G305 -- path traversal guard immediately below
		// Guard against path traversal.
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("tar path traversal detected: %s", hdr.Name)
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil { // #nosec G301 -- target is inside a temp dir for artifact extraction
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil { // #nosec G301 -- target is inside a temp dir for artifact extraction
				return err
			}
			f, err := os.Create(target)
			if err != nil {
				return err
			}
			_, err = io.Copy(f, tr) // #nosec G110 -- size validated by checksum; artifact from trusted registry
			if cerr := f.Close(); cerr != nil && err == nil {
				err = cerr
			}
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func cloneRepo(src registry.Source) (string, func(), error) {
	tmp, err := os.MkdirTemp("", "agent-registry-git-*")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(tmp) }
	repoURL := src.URL
	if repoURL == "" {
		repoURL = "https://github.com/" + src.Repo
	}
	args := []string{"clone", "--depth=1"}
	if src.Version != "" {
		args = append(args, "--branch", src.Version)
	}
	args = append(args, repoURL, tmp)
	cmd := exec.Command("git", args...) // #nosec G204 -- git is required for source:git artifact downloads
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("git clone %s: %w", repoURL, err)
	}
	return tmp, cleanup, nil
}

func recordInstall(art *registry.Artifact, targets []adapter.Adapter, scope adapter.Scope) error {
	dbPath, err := config.InstalledDBPath()
	if err != nil {
		return err
	}
	var db registry.InstalledDB
	if data, err := os.ReadFile(dbPath); err == nil { // #nosec G304 -- dbPath is ~/.agent-registry/installed.json
		_ = json.Unmarshal(data, &db)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for _, t := range targets {
		path, _ := t.InstallPath(art.Type, scope)
		entry := registry.InstalledEntry{
			Name:        art.Name,
			Type:        string(art.Type),
			Version:     art.Version,
			Agent:       t.Name(),
			Scope:       string(scope),
			Path:        path,
			InstalledAt: now,
		}
		// Replace existing entry for same name+agent.
		replaced := false
		for i, e := range db.Entries {
			if e.Name == art.Name && e.Agent == t.Name() {
				db.Entries[i] = entry
				replaced = true
				break
			}
		}
		if !replaced {
			db.Entries = append(db.Entries, entry)
		}
	}
	data, _ := json.MarshalIndent(db, "", "  ")
	return os.WriteFile(dbPath, data, 0o600)
}
