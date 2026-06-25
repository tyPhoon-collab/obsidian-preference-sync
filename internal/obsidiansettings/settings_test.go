package obsidiansettings

import (
	"os"
	"path/filepath"
	"testing"

	"obsidian-preference-sync/internal/vault"
)

func TestPlanRejectsUnsupportedSetting(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".obsidian"), 0o755); err != nil {
		t.Fatal(err)
	}
	v, err := vault.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Plan(v, "workspace", filepath.Join(root, "workspace.json")); err == nil {
		t.Fatal("expected unsupported setting error")
	}
}

func TestApplyCopiesHotkeysToObsidianDir(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".obsidian"), 0o755); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "hotkeys.json")
	if err := os.WriteFile(source, []byte("{\"x\":[]}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	v, err := vault.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	cp, err := Plan(v, "hotkeys", source)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(cp, false); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(root, ".obsidian", "hotkeys.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "{\"x\":[]}\n" {
		t.Fatalf("got %q", got)
	}
}

func TestApplyCopiesCommandPaletteToObsidianDir(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".obsidian"), 0o755); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "command-palette.json")
	if err := os.WriteFile(source, []byte("{\"pinned\":[\"daily-notes\"]}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	v, err := vault.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	cp, err := Plan(v, "command-palette", source)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(cp, false); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(root, ".obsidian", "command-palette.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "{\"pinned\":[\"daily-notes\"]}\n" {
		t.Fatalf("got %q", got)
	}
}

func TestPlanMarksUnchangedHotkeys(t *testing.T) {
	root := t.TempDir()
	obsidianDir := filepath.Join(root, ".obsidian")
	if err := os.MkdirAll(obsidianDir, 0o755); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "hotkeys.json")
	content := []byte("{\"x\":[]}\n")
	if err := os.WriteFile(source, content, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(obsidianDir, "hotkeys.json"), content, 0o644); err != nil {
		t.Fatal(err)
	}
	v, err := vault.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	cp, err := Plan(v, "hotkeys", source)
	if err != nil {
		t.Fatal(err)
	}
	if cp.Changed {
		t.Fatal("expected unchanged hotkeys")
	}
}
