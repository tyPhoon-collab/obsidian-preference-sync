package appsettings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"obsidian-preference-sync/internal/vault"
)

func TestApplyVimModeUpdatesOnlyVimMode(t *testing.T) {
	root := t.TempDir()
	obsidianDir := filepath.Join(root, ".obsidian")
	if err := os.MkdirAll(obsidianDir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(obsidianDir, "app.json")
	if err := os.WriteFile(path, []byte(`{"promptDelete":false,"vimMode":false,"readableLineLength":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	v, err := vault.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := BuildVimModePlan(v, true)
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Changed {
		t.Fatal("expected changed plan")
	}
	if err := ApplyVimMode(plan); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got["vimMode"] != true {
		t.Fatalf("got vimMode %v", got["vimMode"])
	}
	if got["promptDelete"] != false || got["readableLineLength"] != true {
		t.Fatalf("unexpected preserved fields: %v", got)
	}
}

func TestApplyShowLineNumberUpdatesOnlyShowLineNumber(t *testing.T) {
	root := t.TempDir()
	obsidianDir := filepath.Join(root, ".obsidian")
	if err := os.MkdirAll(obsidianDir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(obsidianDir, "app.json")
	if err := os.WriteFile(path, []byte(`{"promptDelete":false,"vimMode":true,"showLineNumber":false}`), 0o644); err != nil {
		t.Fatal(err)
	}
	v, err := vault.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := BuildShowLineNumberPlan(v, true)
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Changed {
		t.Fatal("expected changed plan")
	}
	if err := ApplyShowLineNumber(plan); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got["showLineNumber"] != true {
		t.Fatalf("got showLineNumber %v", got["showLineNumber"])
	}
	if got["promptDelete"] != false || got["vimMode"] != true {
		t.Fatalf("unexpected preserved fields: %v", got)
	}
}

func TestBuildVimModePlanNoChange(t *testing.T) {
	root := t.TempDir()
	obsidianDir := filepath.Join(root, ".obsidian")
	if err := os.MkdirAll(obsidianDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(obsidianDir, "app.json"), []byte(`{"vimMode":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	v, err := vault.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := BuildVimModePlan(v, true)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Changed {
		t.Fatal("expected no change")
	}
}

func TestBuildShowLineNumberPlanNoChange(t *testing.T) {
	root := t.TempDir()
	obsidianDir := filepath.Join(root, ".obsidian")
	if err := os.MkdirAll(obsidianDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(obsidianDir, "app.json"), []byte(`{"showLineNumber":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	v, err := vault.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := BuildShowLineNumberPlan(v, true)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Changed {
		t.Fatal("expected no change")
	}
}
