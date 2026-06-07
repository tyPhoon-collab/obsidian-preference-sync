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
	VaultFiles     []FileCopy        `toml:"vault_files"`
	PluginSettings map[string]string `toml:"plugin_settings"`
}

type FileCopy struct {
	Source string `toml:"source"`
	Target string `toml:"target"`
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
	for i := range cfg.VaultFiles {
		expanded, err := ExpandPath(cfg.VaultFiles[i].Source, base)
		if err != nil {
			return Config{}, fmt.Errorf("vault_files[%d].source: %w", i, err)
		}
		cfg.VaultFiles[i].Source = expanded
		cfg.VaultFiles[i].Target = filepath.Clean(cfg.VaultFiles[i].Target)
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
	seenVaultFiles := map[string]bool{}
	for i, file := range c.VaultFiles {
		if strings.TrimSpace(file.Source) == "" {
			return fmt.Errorf("vault_files[%d].source is empty", i)
		}
		if strings.TrimSpace(file.Target) == "" {
			return fmt.Errorf("vault_files[%d].target is empty", i)
		}
		target := filepath.Clean(file.Target)
		if filepath.IsAbs(target) || target == "." || target == ".." || strings.HasPrefix(target, ".."+string(filepath.Separator)) {
			return fmt.Errorf("vault_files[%d].target must be a vault-relative file path", i)
		}
		if seenVaultFiles[target] {
			return fmt.Errorf("duplicate vault_files target %q", target)
		}
		seenVaultFiles[target] = true
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
