package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadResolvesRelativePluginSettingsFromConfigDir(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(configPath, []byte(`
plugins = ["obsidian-linter"]

[plugin_settings]
obsidian-linter = "plugin-settings/obsidian-linter"
`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "plugin-settings", "obsidian-linter")
	if cfg.PluginSettings["obsidian-linter"] != want {
		t.Fatalf("got %q, want %q", cfg.PluginSettings["obsidian-linter"], want)
	}
}

func TestLoadResolvesTopLevelHotkeysFromConfigDir(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(configPath, []byte(`
plugins = []
hotkeys = "obsidian-settings/hotkeys.json"
`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "obsidian-settings", "hotkeys.json")
	if cfg.Hotkeys != want {
		t.Fatalf("got %q, want %q", cfg.Hotkeys, want)
	}
}

func TestValidateRejectsDuplicatePluginIDs(t *testing.T) {
	cfg := Config{Plugins: []string{"a", "a"}}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected duplicate plugin id error")
	}
}

func TestValidateRejectsDuplicateThemes(t *testing.T) {
	cfg := Config{Themes: []string{"Primary", "Primary"}}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected duplicate theme error")
	}
}

func TestValidateRequiresActiveThemeInThemes(t *testing.T) {
	cfg := Config{Themes: []string{"Minimal"}, ActiveTheme: "Primary"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected active theme validation error")
	}
}

func TestDangerousSettingsIDs(t *testing.T) {
	got := DangerousSettingsIDs(map[string]string{
		"obsidian-linter": "",
		"copilot":         "",
		"obsidian-git":    "",
	})
	want := []string{"copilot", "obsidian-git"}
	if !reflect.DeepEqual(got, want) && !reflect.DeepEqual(got, []string{"obsidian-git", "copilot"}) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
