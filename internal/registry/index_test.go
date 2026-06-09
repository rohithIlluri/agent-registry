package registry_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/rohithilluri/agent-registry/internal/registry"
)

func sampleIndex() *registry.Index {
	return &registry.Index{
		Version:   "1",
		Generated: "2026-05-30T00:00:00Z",
		Artifacts: []registry.IndexEntry{
			{
				Name:        "io.github.example/filesystem",
				Type:        registry.TypeMCPServer,
				Version:     "1.0.0",
				Description: "Read and write local files",
				Category:    "devops",
				Keywords:    []string{"files", "filesystem", "read"},
				AgentCompat: []string{"claude-code", "codex"},
				Trust:       registry.TrustCurated,
				HasMCP:      true,
			},
			{
				Name:        "io.github.example/code-review",
				Type:        registry.TypeSkill,
				Version:     "1.0.0",
				Description: "Systematic code review with OWASP coverage",
				Category:    "security",
				Keywords:    []string{"code-review", "security", "owasp"},
				AgentCompat: []string{"claude-code", "codex"},
				Trust:       registry.TrustVerified,
			},
			{
				Name:        "io.github.example/debug-cmd",
				Type:        registry.TypeSlashCommand,
				Version:     "1.0.0",
				Description: "Debug slash command",
				Category:    "productivity",
				Keywords:    []string{"debug"},
				AgentCompat: []string{"claude-code"},
				Trust:       registry.TrustCommunity,
			},
		},
	}
}

func TestSearch_EmptyQuery(t *testing.T) {
	idx := sampleIndex()
	results := registry.Search(idx, "", "", "", "")
	if len(results) != 3 {
		t.Errorf("expected 3 results for empty query, got %d", len(results))
	}
}

func TestSearch_ByName(t *testing.T) {
	idx := sampleIndex()
	results := registry.Search(idx, "filesystem", "", "", "")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Name != "io.github.example/filesystem" {
		t.Errorf("unexpected result: %s", results[0].Name)
	}
}

func TestSearch_ByKeyword(t *testing.T) {
	idx := sampleIndex()
	results := registry.Search(idx, "owasp", "", "", "")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Name != "io.github.example/code-review" {
		t.Errorf("unexpected result: %s", results[0].Name)
	}
}

func TestSearch_TypeFilter(t *testing.T) {
	idx := sampleIndex()
	results := registry.Search(idx, "", "skill", "", "")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Type != registry.TypeSkill {
		t.Errorf("wrong type: %s", results[0].Type)
	}
}

func TestSearch_AgentFilter(t *testing.T) {
	idx := sampleIndex()
	// The slash-command only supports claude-code.
	results := registry.Search(idx, "", "", "codex", "")
	for _, r := range results {
		found := false
		for _, a := range r.AgentCompat {
			if a == "codex" {
				found = true
			}
		}
		if !found {
			t.Errorf("result %s is not compatible with codex", r.Name)
		}
	}
	if len(results) != 2 {
		t.Errorf("expected 2 codex-compatible results, got %d", len(results))
	}
}

func TestSearch_CategoryFilter(t *testing.T) {
	idx := sampleIndex()
	results := registry.Search(idx, "", "", "", "security")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Category != "security" {
		t.Errorf("wrong category: %s", results[0].Category)
	}
}

func TestSearch_CaseInsensitive(t *testing.T) {
	idx := sampleIndex()
	results := registry.Search(idx, "FILESYSTEM", "", "", "")
	if len(results) != 1 {
		t.Errorf("expected case-insensitive match, got %d results", len(results))
	}
}

func TestSearch_NoMatch(t *testing.T) {
	idx := sampleIndex()
	results := registry.Search(idx, "xyzzy-not-found", "", "", "")
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestFindInIndex(t *testing.T) {
	idx := sampleIndex()
	entry, ok := registry.FindInIndex(idx, "io.github.example/filesystem")
	if !ok {
		t.Fatal("expected to find artifact")
	}
	if entry.Name != "io.github.example/filesystem" {
		t.Errorf("wrong entry returned: %s", entry.Name)
	}

	_, ok = registry.FindInIndex(idx, "io.github.example/nonexistent")
	if ok {
		t.Error("expected not found")
	}
}

func TestClient_LoadIndex_FromFile(t *testing.T) {
	idx := sampleIndex()
	data, _ := json.Marshal(idx)

	tmp := t.TempDir()
	path := filepath.Join(tmp, "index.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	client := registry.NewClient()
	client.LocalPath = path

	loaded, err := client.LoadIndex()
	if err != nil {
		t.Fatalf("LoadIndex: %v", err)
	}
	if len(loaded.Artifacts) != 3 {
		t.Errorf("expected 3 artifacts, got %d", len(loaded.Artifacts))
	}
}

func TestClient_LoadIndex_FromHTTP(t *testing.T) {
	idx := sampleIndex()
	data, _ := json.Marshal(idx)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(data) //nolint:errcheck
	}))
	defer srv.Close()

	client := &registry.Client{
		IndexURL: srv.URL,
		CacheDir: t.TempDir(),
	}
	loaded, err := client.LoadIndex()
	if err != nil {
		t.Fatalf("LoadIndex: %v", err)
	}
	if len(loaded.Artifacts) != 3 {
		t.Errorf("expected 3 artifacts, got %d", len(loaded.Artifacts))
	}
}

func TestClient_LoadArtifact_FromHTTP(t *testing.T) {
	art := &registry.Artifact{
		Name:        "io.github.example/filesystem",
		Type:        registry.TypeMCPServer,
		Version:     "1.0.0",
		Description: "Read and write local files",
		Category:    "devops",
		AgentCompat: []string{"claude-code", "codex"},
		Trust:       registry.TrustCurated,
	}
	data, _ := json.Marshal(art)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(data) //nolint:errcheck
	}))
	defer srv.Close()

	client := &registry.Client{
		IndexURL: srv.URL + "/index.json",
		CacheDir: t.TempDir(),
	}
	loaded, err := client.LoadArtifact("io.github.example/filesystem")
	if err != nil {
		t.Fatalf("LoadArtifact: %v", err)
	}
	if loaded.Name != art.Name {
		t.Errorf("wrong artifact: %s", loaded.Name)
	}
}
