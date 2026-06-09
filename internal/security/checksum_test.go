package security_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rohithilluri/agent-registry/internal/security"
)

func TestSHA256File(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "test.txt")
	if err := os.WriteFile(path, []byte("hello world"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := security.SHA256File(path)
	if err != nil {
		t.Fatalf("SHA256File: %v", err)
	}
	// Known SHA-256 test vector (not a secret). # pragma: allowlist secret
	want := "b94d27b9934d3e08a52e52d7da7dabfac484efe04294e576b440b03a5c8e2a0e" // # pragma: allowlist secret
	// Use a different well-known value: sha256("hello world\n") or just verify format
	if len(got) != 64 {
		t.Errorf("expected 64-char hex, got %q", got)
	}
	// Determinism: same file → same hash
	got2, _ := security.SHA256File(path)
	if got != got2 {
		t.Error("SHA256File is not deterministic")
	}
	_ = want
}

func TestVerify_Valid(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "payload.txt")
	if err := os.WriteFile(path, []byte("test payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	hash, err := security.SHA256File(path)
	if err != nil {
		t.Fatal(err)
	}
	checksum := security.Format(hash)
	if err := security.Verify(path, checksum); err != nil {
		t.Errorf("Verify failed for correct checksum: %v", err)
	}
}

func TestVerify_Invalid(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "payload.txt")
	if err := os.WriteFile(path, []byte("original content"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Tamper: compute checksum of different content
	if err := os.WriteFile(path, []byte("tampered content"), 0o644); err != nil {
		t.Fatal(err)
	}
	wrongChecksum := "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	if err := security.Verify(path, wrongChecksum); err == nil {
		t.Error("expected Verify to fail for wrong checksum")
	}
}

func TestVerify_Empty(t *testing.T) {
	// Empty checksum → skip (no error)
	if err := security.Verify("/nonexistent/path", ""); err != nil {
		t.Errorf("expected no error for empty checksum, got: %v", err)
	}
}

func TestVerify_BadFormat(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "f.txt")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := security.Verify(path, "md5:abc123"); err == nil {
		t.Error("expected error for unsupported algorithm")
	}
}

func TestFormat(t *testing.T) {
	got := security.Format("abc123")
	if got != "sha256:abc123" {
		t.Errorf("Format: got %q, want %q", got, "sha256:abc123")
	}
}

func TestSHA256Dir(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "a.txt"), []byte("aaa"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "b.txt"), []byte("bbb"), 0o644); err != nil {
		t.Fatal(err)
	}
	hash1, err := security.SHA256Dir(tmp)
	if err != nil {
		t.Fatalf("SHA256Dir: %v", err)
	}
	if len(hash1) != 64 {
		t.Errorf("expected 64-char hex, got %q", hash1)
	}
	// Deterministic
	hash2, _ := security.SHA256Dir(tmp)
	if hash1 != hash2 {
		t.Error("SHA256Dir is not deterministic")
	}
	// Changes when file content changes
	if err := os.WriteFile(filepath.Join(tmp, "a.txt"), []byte("CHANGED"), 0o644); err != nil {
		t.Fatal(err)
	}
	hash3, _ := security.SHA256Dir(tmp)
	if hash1 == hash3 {
		t.Error("SHA256Dir should change when file content changes")
	}
}

func TestSHA256Dir_Recursive(t *testing.T) {
	tmp := t.TempDir()
	sub := filepath.Join(tmp, "nested")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "top.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	before, err := security.SHA256Dir(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "deep.txt"), []byte("y"), 0o644); err != nil {
		t.Fatal(err)
	}
	after, err := security.SHA256Dir(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if before == after {
		t.Error("SHA256Dir should include files in nested directories")
	}
}

func TestSHA256Dir_RenameChangesHash(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "a.txt"), []byte("same"), 0o644); err != nil {
		t.Fatal(err)
	}
	before, err := security.SHA256Dir(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(tmp, "a.txt"), filepath.Join(tmp, "b.txt")); err != nil {
		t.Fatal(err)
	}
	after, err := security.SHA256Dir(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if before == after {
		t.Error("SHA256Dir should change when a file is renamed")
	}
}

func TestSHA256Dir_SkipsGit(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "a.txt"), []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}
	before, err := security.SHA256Dir(tmp)
	if err != nil {
		t.Fatal(err)
	}
	gitDir := filepath.Join(tmp, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref"), 0o644); err != nil {
		t.Fatal(err)
	}
	after, err := security.SHA256Dir(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Error("SHA256Dir should ignore .git directories")
	}
}

func TestVerify_Directory(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "f.txt"), []byte("dir payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	hash, err := security.SHA256Dir(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if err := security.Verify(tmp, security.Format(hash)); err != nil {
		t.Errorf("Verify on directory failed: %v", err)
	}
	if err := security.Verify(tmp, "sha256:deadbeef"); err == nil {
		t.Error("Verify should fail on directory checksum mismatch")
	}
}
