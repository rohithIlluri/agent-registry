package installer

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

// makeTarGz builds an in-memory tar.gz with the given path→content entries.
func makeTarGz(t *testing.T, files map[string]string) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, content := range files {
		hdr := &tar.Header{Name: name, Mode: 0o644, Size: int64(len(content))}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return &buf
}

func TestUnpackTarGz(t *testing.T) {
	dest := t.TempDir()
	archive := makeTarGz(t, map[string]string{
		"repo-main/skills/code-review/SKILL.md": "# skill",
		"repo-main/README.md":                   "readme",
	})
	if err := unpackTarGz(archive, dest); err != nil {
		t.Fatalf("unpackTarGz: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dest, "repo-main", "skills", "code-review", "SKILL.md"))
	if err != nil {
		t.Fatalf("expected SKILL.md to be extracted: %v", err)
	}
	if string(data) != "# skill" {
		t.Errorf("SKILL.md content = %q", data)
	}
}

func TestUnpackTarGz_RejectsTraversal(t *testing.T) {
	dest := t.TempDir()
	archive := makeTarGz(t, map[string]string{"../escape.txt": "evil"})
	if err := unpackTarGz(archive, dest); err == nil {
		t.Fatal("expected path traversal error, got nil")
	}
}

func TestResolveSubPath_GitHubArchiveWrapper(t *testing.T) {
	// GitHub tarballs wrap content in a single "<repo>-<ref>/" directory.
	root := t.TempDir()
	skillDir := filepath.Join(root, "agent-registry-main", "skills", "code-review")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := resolveSubPath(root, "skills/code-review")
	if err != nil {
		t.Fatalf("resolveSubPath: %v", err)
	}
	if got != skillDir {
		t.Errorf("resolveSubPath = %q, want %q", got, skillDir)
	}
}

func TestResolveSubPath_TopLevel(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "skills", "code-review")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// A second top-level entry prevents wrapper-dir descent.
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := resolveSubPath(root, "skills/code-review")
	if err != nil {
		t.Fatalf("resolveSubPath: %v", err)
	}
	if got != skillDir {
		t.Errorf("resolveSubPath = %q, want %q", got, skillDir)
	}
}

func TestResolveSubPath_NotFound(t *testing.T) {
	root := t.TempDir()
	if _, err := resolveSubPath(root, "skills/missing"); err == nil {
		t.Fatal("expected error for missing skillPath")
	}
}

func TestResolveSubPath_RejectsEscape(t *testing.T) {
	root := t.TempDir()
	if _, err := resolveSubPath(root, "../outside"); err == nil {
		t.Fatal("expected error for escaping skillPath")
	}
}

func TestResolveSubPath_EmptySubReturnsRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "b.txt"), []byte("y"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := resolveSubPath(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if got != root {
		t.Errorf("resolveSubPath(root, \"\") = %q, want %q", got, root)
	}
}

func TestDescendSingleDir(t *testing.T) {
	root := t.TempDir()
	inner := filepath.Join(root, "wrapper")
	if err := os.MkdirAll(inner, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := descendSingleDir(root); got != inner {
		t.Errorf("descendSingleDir = %q, want %q", got, inner)
	}
}
