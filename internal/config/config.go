package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Plugins        []string          `toml:"plugins"`
	Themes         []string          `toml:"themes"`
	ActiveTheme    string            `toml:"active_theme"`
	VimMode        *bool             `toml:"vim_mode"`
	Hotkeys        string            `toml:"hotkeys"`
	Vimrc          string            `toml:"vimrc"`
	PluginSettings map[string]string `toml:"plugin_settings"`
}

func Load(path string) (Config, error) {
	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return Config{}, fmt.Errorf("read config %s: %w", path, err)
	}
	if cfg.PluginSettings == nil {
		cfg.PluginSettings = map[string]string{}
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	base := filepath.Dir(path)
	for pluginID, source := range cfg.PluginSettings {
		expanded, err := ExpandPath(source, base)
		if err != nil {
			return Config{}, fmt.Errorf("plugin_settings.%s: %w", pluginID, err)
		}
		cfg.PluginSettings[pluginID] = expanded
	}
	if cfg.Hotkeys != "" {
		expanded, err := ExpandPath(cfg.Hotkeys, base)
		if err != nil {
			return Config{}, fmt.Errorf("hotkeys: %w", err)
		}
		cfg.Hotkeys = expanded
	}
	if cfg.Vimrc != "" {
		expanded, err := ExpandPath(cfg.Vimrc, base)
		if err != nil {
			return Config{}, fmt.Errorf("vimrc: %w", err)
		}
		cfg.Vimrc = expanded
	}
	return cfg, nil
}

func (c Config) Validate() error {
	seen := map[string]bool{}
	for _, id := range c.Plugins {
		if !ValidPluginID(id) {
			return fmt.Errorf("invalid plugin id %q", id)
		}
		if seen[id] {
			return fmt.Errorf("duplicate plugin id %q", id)
		}
		seen[id] = true
	}
	seenThemes := map[string]bool{}
	for _, name := range c.Themes {
		if strings.TrimSpace(name) == "" || strings.TrimSpace(name) != name {
			return fmt.Errorf("invalid theme name %q", name)
		}
		if seenThemes[name] {
			return fmt.Errorf("duplicate theme name %q", name)
		}
		seenThemes[name] = true
	}
	if strings.TrimSpace(c.ActiveTheme) != c.ActiveTheme {
		return fmt.Errorf("invalid active_theme %q", c.ActiveTheme)
	}
	if c.ActiveTheme != "" && !seenThemes[c.ActiveTheme] {
		return fmt.Errorf("active_theme %q must also be listed in themes", c.ActiveTheme)
	}
	for id, source := range c.PluginSettings {
		if !ValidPluginID(id) {
			return fmt.Errorf("invalid plugin_settings id %q", id)
		}
		if strings.TrimSpace(source) == "" {
			return fmt.Errorf("plugin_settings.%s has empty source path", id)
		}
	}
	return nil
}

func (c Config) PluginIDs() []string {
	ids := append([]string(nil), c.Plugins...)
	sort.Strings(ids)
	return ids
}

func ValidPluginID(id string) bool {
	if id == "" || strings.TrimSpace(id) != id {
		return false
	}
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			continue
		}
		return false
	}
	return true
}

func ExpandPath(path string, baseDir string) (string, error) {
	if path == "" {
		return "", nil
	}
	if strings.HasPrefix(path, "~/") || path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if path == "~" {
			return home, nil
		}
		return filepath.Join(home, strings.TrimPrefix(path, "~/")), nil
	}
	if filepath.IsAbs(path) {
		return filepath.Clean(path), nil
	}
	return filepath.Clean(filepath.Join(baseDir, path)), nil
}
