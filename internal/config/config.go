package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds persistent CLI configuration.
type Config struct {
	IndexURL     string `json:"indexURL,omitempty"`     // override default registry URL
	DefaultAgent string `json:"defaultAgent,omitempty"` // "claude-code" | "codex"
	AutoConfirm  bool   `json:"autoConfirm,omitempty"`  // skip install confirmation
}

// dir returns the config directory, creating it if absent.
func dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	d := filepath.Join(home, ".agent-registry")
	return d, os.MkdirAll(d, 0o700)
}

func configPath() (string, error) {
	d, err := dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "config.json"), nil
}

// Load reads the config file, returning defaults if absent.
func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path) // #nosec G304 -- path is ~/.agent-registry/config.json, not user-supplied
	if os.IsNotExist(err) {
		return &Config{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}

// Save writes the config to disk.
func Save(cfg *Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// InstalledDBPath returns the path to the local installed-artifact database.
func InstalledDBPath() (string, error) {
	d, err := dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "installed.json"), nil
}
