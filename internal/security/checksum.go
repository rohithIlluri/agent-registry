package security

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Verify checks that the file or directory at path matches the expected
// checksum. expected format: "sha256:<hex>"
func Verify(path, expected string) error {
	if expected == "" {
		return nil // no checksum recorded; skip
	}
	algo, want, ok := strings.Cut(expected, ":")
	if !ok || algo != "sha256" {
		return fmt.Errorf("unsupported checksum format %q (want sha256:<hex>)", expected)
	}
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}
	var got string
	if fi.IsDir() {
		got, err = SHA256Dir(path)
	} else {
		got, err = SHA256File(path)
	}
	if err != nil {
		return err
	}
	if got != want {
		return fmt.Errorf("checksum mismatch: got sha256:%s, want sha256:%s", got, want)
	}
	return nil
}

// SHA256File returns the lowercase hex SHA-256 of a file.
func SHA256File(path string) (string, error) {
	f, err := os.Open(path) // #nosec G304 -- path is a registry artifact being verified
	if err != nil {
		return "", fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()
	return SHA256Reader(f)
}

// SHA256Reader returns the lowercase hex SHA-256 of a reader.
func SHA256Reader(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", fmt.Errorf("hash: %w", err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// SHA256Dir returns a deterministic SHA-256 over all files under dir.
// Files are walked recursively in sorted order; each contributes its
// slash-separated relative path, a NUL separator, and its contents, so both
// renames and content changes alter the hash. `.git` directories are skipped
// because clone metadata is not part of the artifact.
func SHA256Dir(dir string) (string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("walk dir %s: %w", dir, err)
	}
	sort.Strings(files)

	h := sha256.New()
	for _, path := range files {
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return "", err
		}
		h.Write([]byte(filepath.ToSlash(rel)))
		h.Write([]byte{0})
		data, err := os.ReadFile(path) // #nosec G304 -- path is inside the artifact dir being hashed
		if err != nil {
			return "", err
		}
		h.Write(data)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Format wraps a raw hex string into the canonical "sha256:<hex>" form.
func Format(hex string) string {
	return "sha256:" + hex
}
