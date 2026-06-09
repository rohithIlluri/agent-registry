package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rohithilluri/agent-registry/internal/config"
	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Get and set agr configuration",
		Long: `Manage persistent agr settings stored in ~/.agent-registry/config.json.

Available keys:
  index-url      URL of the registry index JSON (default: GitHub raw URL)
  default-agent  Default agent target: claude-code | codex
  auto-confirm   Skip install confirmation prompts: true | false`,
	}
	cmd.AddCommand(newConfigGetCmd(), newConfigSetCmd(), newConfigListCmd())
	return cmd
}

func newConfigGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Print a single config value",
		Args:  cobra.ExactArgs(1),
		Example: `  agr config get index-url
  agr config get default-agent`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			val, err := getField(cfg, args[0])
			if err != nil {
				return err
			}
			fmt.Println(val)
			return nil
		},
	}
}

func newConfigSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a config value",
		Args:  cobra.ExactArgs(2),
		Example: `  agr config set default-agent codex
  agr config set auto-confirm true
  agr config set index-url https://example.com/my-index.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if err := setField(cfg, args[0], args[1]); err != nil {
				return err
			}
			if err := config.Save(cfg); err != nil {
				return err
			}
			fmt.Printf("Set %s = %s\n", args[0], args[1])
			return nil
		},
	}
}

func newConfigListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "Print all config values",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			data, err := json.MarshalIndent(cfg, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		},
	}
}

func getField(cfg *config.Config, key string) (string, error) {
	switch strings.ToLower(key) {
	case "index-url":
		if cfg.IndexURL == "" {
			return "(default)", nil
		}
		return cfg.IndexURL, nil
	case "default-agent":
		if cfg.DefaultAgent == "" {
			return "(auto-detect)", nil
		}
		return cfg.DefaultAgent, nil
	case "auto-confirm":
		return fmt.Sprintf("%v", cfg.AutoConfirm), nil
	default:
		return "", fmt.Errorf("unknown config key %q; valid keys: index-url, default-agent, auto-confirm", key)
	}
}

func setField(cfg *config.Config, key, val string) error {
	switch strings.ToLower(key) {
	case "index-url":
		cfg.IndexURL = val
	case "default-agent":
		if val != "claude-code" && val != "codex" && val != "" {
			return fmt.Errorf("invalid agent %q; use: claude-code, codex, or empty string to auto-detect", val)
		}
		cfg.DefaultAgent = val
	case "auto-confirm":
		switch strings.ToLower(val) {
		case "true", "1", "yes":
			cfg.AutoConfirm = true
		case "false", "0", "no", "":
			cfg.AutoConfirm = false
		default:
			return fmt.Errorf("invalid boolean %q; use: true or false", val)
		}
	default:
		return fmt.Errorf("unknown config key %q; valid keys: index-url, default-agent, auto-confirm", key)
	}
	return nil
}
