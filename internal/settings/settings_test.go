package settings

import (
	"os"
	"path/filepath"
	"testing"

	"obsidian-preference-sync/internal/vault"
)

func TestSettingsCopyExcludesPluginRuntimeFiles(t *testing.T) {
	root := t.TempDir()
	obsidianDir := filepath.Join(root, ".obsidian")
	pluginDir := filepath.Join(obsidianDir, "plugins", "obsidian-linter")
	sourceDir := filepath.Join(root, "source")
	for _, dir := range []string{
		pluginDir,
		sourceDir,
		filepath.Join(sourceDir, "node_modules"),
		filepath.Join(sourceDir, "nested"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	files := map[string]string{
		filepath.Join(pluginDir, "manifest.json"):        "{}",
		filepath.Join(pluginDir, "main.js"):              "",
		filepath.Join(sourceDir, "data.json"):            "{}",
		filepath.Join(sourceDir, "main.js"):              "bad",
		filepath.Join(sourceDir, "manifest.json"):        "bad",
		filepath.Join(sourceDir, "styles.css"):           "bad",
		filepath.Join(sourceDir, "node_modules", "x.js"): "bad",
		filepath.Join(sourceDir, "nested", "prefs.json"): "{\"ok\":true}",
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	v, err := vault.Open(root)
	if err != nil {
		t.Fatal(err)
	}

	cp, err := Plan(v, "obsidian-linter", sourceDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(cp, false); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(pluginDir, "data.json")); err != nil {
		t.Fatal("expected data.json copied")
	}
	if _, err := os.Stat(filepath.Join(pluginDir, "nested", "prefs.json")); err != nil {
		t.Fatal("expected nested prefs copied")
	}
	if got, err := os.ReadFile(filepath.Join(pluginDir, "main.js")); err != nil || string(got) != "" {
		t.Fatalf("runtime main.js should not be overwritten, got %q err=%v", got, err)
	}
	if _, err := os.Stat(filepath.Join(pluginDir, "node_modules", "x.js")); !os.IsNotExist(err) {
		t.Fatal("node_modules should not be copied")
	}
}

func TestPlanOnlyIncludesChangedFiles(t *testing.T) {
	root := t.TempDir()
	pluginDir := filepath.Join(root, ".obsidian", "plugins", "obsidian-linter")
	sourceDir := filepath.Join(root, "source")
	for _, dir := range []string{pluginDir, sourceDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	files := map[string]string{
		filepath.Join(pluginDir, "manifest.json"): "{}",
		filepath.Join(pluginDir, "main.js"):       "",
		filepath.Join(sourceDir, "same.json"):     "{\"same\":true}\n",
		filepath.Join(pluginDir, "same.json"):     "{\"same\":true}\n",
		filepath.Join(sourceDir, "changed.json"):  "{\"source\":true}\n",
		filepath.Join(pluginDir, "changed.json"):  "{\"target\":true}\n",
		filepath.Join(sourceDir, "new.json"):      "{\"new\":true}\n",
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	v, err := vault.Open(root)
	if err != nil {
		t.Fatal(err)
	}

	cp, err := Plan(v, "obsidian-linter", sourceDir)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"changed.json": true, "new.json": true}
	if len(cp.Files) != len(want) {
		t.Fatalf("got files %v, want changed and new only", cp.Files)
	}
	for _, file := range cp.Files {
		if !want[file] {
			t.Fatalf("unexpected planned file %q in %v", file, cp.Files)
		}
	}
}
