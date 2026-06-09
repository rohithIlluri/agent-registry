package registry

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	DefaultIndexURL = "https://raw.githubusercontent.com/rohithilluri/agent-registry/main/registry/index.json"
	CacheTTL        = 24 * time.Hour
	cacheFile       = "index.json"
	httpTimeout     = 30 * time.Second
)

// httpClient is the shared client used for all registry requests.
// A 30 s timeout prevents the CLI from hanging on a slow or adversarial server.
var httpClient = &http.Client{Timeout: httpTimeout}

// Client fetches and caches the registry index.
type Client struct {
	IndexURL  string
	CacheDir  string
	LocalPath string // non-empty → use local index (for development / self-hosting)
}

// NewClient returns a Client using the default index URL and the system cache dir.
func NewClient() *Client {
	cacheDir := filepath.Join(cacheBaseDir(), "agent-registry")
	return &Client{
		IndexURL: DefaultIndexURL,
		CacheDir: cacheDir,
	}
}

func cacheBaseDir() string {
	if d, ok := os.LookupEnv("XDG_CACHE_HOME"); ok && d != "" {
		return d
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache")
}

// LoadIndex returns the index, using a cached copy if fresh.
// If a local registry path is set it always reads from there.
func (c *Client) LoadIndex() (*Index, error) {
	if c.LocalPath != "" {
		return c.loadFromFile(c.LocalPath)
	}
	cached := filepath.Join(c.CacheDir, cacheFile)
	if fi, err := os.Stat(cached); err == nil {
		if time.Since(fi.ModTime()) < CacheTTL {
			return c.loadFromFile(cached)
		}
	}
	return c.fetchAndCache(cached)
}

func (c *Client) loadFromFile(path string) (*Index, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- path is a cache file or local index path, not user-supplied
	if err != nil {
		return nil, fmt.Errorf("read index %s: %w", path, err)
	}
	var idx Index
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("parse index: %w", err)
	}
	return &idx, nil
}

func (c *Client) fetchAndCache(dest string) (*Index, error) {
	resp, err := httpClient.Get(c.IndexURL) // #nosec G107 -- URL is user-configurable or the default registry URL
	if err != nil {
		return nil, fmt.Errorf("fetch index from %s: %w", c.IndexURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch index: HTTP %d from %s", resp.StatusCode, c.IndexURL)
	}
	var idx Index
	if err := json.NewDecoder(resp.Body).Decode(&idx); err != nil {
		return nil, fmt.Errorf("decode index: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o750); err == nil {
		data, _ := json.MarshalIndent(idx, "", "  ")
		_ = os.WriteFile(dest, data, 0o644) // #nosec G306 -- registry index cache is a public data file
	}
	return &idx, nil
}

// Search returns index entries matching the query across name, description, and keywords.
// Optional filters: artifactType (empty = all), agentCompat (empty = all), category (empty = all).
func Search(idx *Index, query, artifactType, agentCompat, category string) []IndexEntry {
	q := strings.ToLower(query)
	var results []IndexEntry
	for _, e := range idx.Artifacts {
		if artifactType != "" && string(e.Type) != artifactType {
			continue
		}
		if category != "" && e.Category != category {
			continue
		}
		if agentCompat != "" && !containsStr(e.AgentCompat, agentCompat) {
			continue
		}
		if q == "" || matchesQuery(e, q) {
			results = append(results, e)
		}
	}
	return results
}

func matchesQuery(e IndexEntry, q string) bool {
	if strings.Contains(strings.ToLower(e.Name), q) {
		return true
	}
	if strings.Contains(strings.ToLower(e.DisplayName), q) {
		return true
	}
	if strings.Contains(strings.ToLower(e.Description), q) {
		return true
	}
	for _, kw := range e.Keywords {
		if strings.Contains(strings.ToLower(kw), q) {
			return true
		}
	}
	return false
}

func containsStr(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

// ArtifactBaseURL derives the per-artifact manifest URL from the index URL.
func ArtifactBaseURL(indexURL string) string {
	// e.g. .../registry/index.json → .../registry/artifacts/
	return strings.Replace(indexURL, "index.json", "artifacts/", 1)
}

// LoadArtifact fetches the full manifest for a named artifact.
// name format: "namespace/artifact-name"
func (c *Client) LoadArtifact(name string) (*Artifact, error) {
	// Try local first
	if c.LocalPath != "" {
		dir := filepath.Dir(c.LocalPath)
		artPath := filepath.Join(dir, "artifacts", name+".json")
		return c.loadArtifactFromFile(artPath)
	}
	baseURL := ArtifactBaseURL(c.IndexURL)
	url := baseURL + name + ".json"
	resp, err := httpClient.Get(url) // #nosec G107 -- URL derived from the registry index URL, not raw user input
	if err != nil {
		return nil, fmt.Errorf("fetch artifact %s: %w", name, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("artifact %q not found in registry", name)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch artifact %s: HTTP %d", name, resp.StatusCode)
	}
	var art Artifact
	if err := json.NewDecoder(resp.Body).Decode(&art); err != nil {
		return nil, fmt.Errorf("decode artifact: %w", err)
	}
	return &art, nil
}

func (c *Client) loadArtifactFromFile(path string) (*Artifact, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- path is a local registry artifacts path, not user-supplied
	if err != nil {
		return nil, fmt.Errorf("read artifact %s: %w", path, err)
	}
	var art Artifact
	if err := json.Unmarshal(data, &art); err != nil {
		return nil, fmt.Errorf("parse artifact: %w", err)
	}
	return &art, nil
}

// FindInIndex looks up an IndexEntry by exact name.
func FindInIndex(idx *Index, name string) (IndexEntry, bool) {
	for _, e := range idx.Artifacts {
		if e.Name == name {
			return e, true
		}
	}
	return IndexEntry{}, false
}
