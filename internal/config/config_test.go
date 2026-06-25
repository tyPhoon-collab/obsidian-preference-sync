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

func TestLoadResolvesTopLevelCommandPaletteFromConfigDir(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(configPath, []byte(`
plugins = []
command_palette = "obsidian-settings/command-palette.json"
`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "obsidian-settings", "command-palette.json")
	if cfg.CommandPalette != want {
		t.Fatalf("got %q, want %q", cfg.CommandPalette, want)
	}
}

func TestLoadVimMode(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(configPath, []byte(`
plugins = []
vim_mode = true
`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.VimMode == nil || *cfg.VimMode != true {
		t.Fatalf("got VimMode %v, want true", cfg.VimMode)
	}
}

func TestLoadShowLineNumber(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(configPath, []byte(`
plugins = []
show_line_number = true
`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ShowLineNumber == nil || *cfg.ShowLineNumber != true {
		t.Fatalf("got ShowLineNumber %v, want true", cfg.ShowLineNumber)
	}
}

func TestLoadExampleConfigIncludesAppSettings(t *testing.T) {
	cfg, err := Load(filepath.Join("..", "..", "examples", "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.VimMode == nil || *cfg.VimMode != true {
		t.Fatalf("got VimMode %v, want true", cfg.VimMode)
	}
	if cfg.ShowLineNumber == nil || *cfg.ShowLineNumber != true {
		t.Fatalf("got ShowLineNumber %v, want true", cfg.ShowLineNumber)
	}
	wantHotkeys := filepath.Join("..", "..", "examples", "obsidian-settings", "hotkeys.json")
	if cfg.Hotkeys != wantHotkeys {
		t.Fatalf("got Hotkeys %q, want %q", cfg.Hotkeys, wantHotkeys)
	}
	wantCommandPalette := filepath.Join("..", "..", "examples", "obsidian-settings", "command-palette.json")
	if cfg.CommandPalette != wantCommandPalette {
		t.Fatalf("got CommandPalette %q, want %q", cfg.CommandPalette, wantCommandPalette)
	}
}

func TestLoadFonts(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(configPath, []byte(`
plugins = []

[fonts]
interface = "Maple Mono NF CN"
text = "Maple Mono NF CN"
monospace = "Maple Mono NF CN"
`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Fonts.Interface != "Maple Mono NF CN" {
		t.Fatalf("got interface font %q", cfg.Fonts.Interface)
	}
	if cfg.Fonts.Text != "Maple Mono NF CN" {
		t.Fatalf("got text font %q", cfg.Fonts.Text)
	}
	if cfg.Fonts.Monospace != "Maple Mono NF CN" {
		t.Fatalf("got monospace font %q", cfg.Fonts.Monospace)
	}
}

func TestLoadResolvesVaultFilesSourceFromConfigDir(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(configPath, []byte(`
plugins = []
[[vault_files]]
source = "vault-files/.obsidian.vimrc"
target = ".obsidian.vimrc"
`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "vault-files", ".obsidian.vimrc")
	if cfg.VaultFiles[0].Source != want {
		t.Fatalf("got %q, want %q", cfg.VaultFiles[0].Source, want)
	}
	if cfg.VaultFiles[0].Target != ".obsidian.vimrc" {
		t.Fatalf("got target %q", cfg.VaultFiles[0].Target)
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

func TestValidateRejectsFontWhitespace(t *testing.T) {
	cfg := Config{Fonts: Fonts{Text: " Maple Mono NF CN"}}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected font whitespace validation error")
	}
}

func TestValidateRejectsUnsafeVaultFileTarget(t *testing.T) {
	cfg := Config{VaultFiles: []FileCopy{{Source: "spacekeys.yml", Target: "../spacekeys.yml"}}}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected unsafe vault file target error")
	}
}

func TestValidateRejectsDuplicateVaultFileTargets(t *testing.T) {
	cfg := Config{VaultFiles: []FileCopy{
		{Source: "a", Target: "spacekeys.yml"},
		{Source: "b", Target: "./spacekeys.yml"},
	}}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected duplicate vault file target error")
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
