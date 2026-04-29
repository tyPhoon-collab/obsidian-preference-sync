package vault

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestUpsertEnabledPluginsPreservesExistingIDs(t *testing.T) {
	root := t.TempDir()
	obsidianDir := filepath.Join(root, ".obsidian")
	if err := os.MkdirAll(obsidianDir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(obsidianDir, "community-plugins.json")
	if err := os.WriteFile(path, []byte("[\"local-only\",\"obsidian-linter\"]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	v, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}

	added, changed, err := v.UpsertEnabledPlugins([]string{"obsidian-linter", "table-editor-obsidian"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected changed")
	}
	if !reflect.DeepEqual(added, []string{"table-editor-obsidian"}) {
		t.Fatalf("got added %v", added)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	want := []string{"local-only", "obsidian-linter", "table-editor-obsidian"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	if data[len(data)-1] != '\n' {
		t.Fatal("expected trailing newline")
	}
}

func TestUpsertEnabledPluginsDryRunDoesNotWrite(t *testing.T) {
	root := t.TempDir()
	obsidianDir := filepath.Join(root, ".obsidian")
	if err := os.MkdirAll(obsidianDir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(obsidianDir, "community-plugins.json")
	original := []byte("[\"local-only\"]\n")
	if err := os.WriteFile(path, original, 0o644); err != nil {
		t.Fatal(err)
	}
	v, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}

	_, changed, err := v.UpsertEnabledPlugins([]string{"obsidian-linter"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected dry-run changed")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(original) {
		t.Fatalf("dry-run wrote file: %q", data)
	}
}
