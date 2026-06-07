package security

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
)

// Verify checks that the file at path matches the expected checksum.
// expected format: "sha256:<hex>"
func Verify(path, expected string) error {
	if expected == "" {
		return nil // no checksum recorded; skip
	}
	algo, want, ok := strings.Cut(expected, ":")
	if !ok || algo != "sha256" {
		return fmt.Errorf("unsupported checksum format %q (want sha256:<hex>)", expected)
	}
	got, err := SHA256File(path)
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
	f, err := os.Open(path)
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

// SHA256Dir returns the SHA-256 of the concatenated sorted file hashes under dir.
func SHA256Dir(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("read dir %s: %w", dir, err)
	}
	h := sha256.New()
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := os.ReadFile(fmt.Sprintf("%s/%s", dir, e.Name())) // #nosec G304 -- dir is a trusted artifact path
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
