package appearance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"obsidian-preference-sync/internal/vault"
)

func TestApplyUpdatesOnlyCSSTheme(t *testing.T) {
	root := t.TempDir()
	obsidianDir := filepath.Join(root, ".obsidian")
	if err := os.MkdirAll(obsidianDir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(obsidianDir, "appearance.json")
	if err := os.WriteFile(path, []byte(`{"accentColor":"red","cssTheme":"Minimal","translucency":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	v, err := vault.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := BuildPlan(v, "Primary")
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Changed {
		t.Fatal("expected changed plan")
	}
	if err := Apply(plan); err != nil {
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
	if got["cssTheme"] != "Primary" {
		t.Fatalf("got cssTheme %v", got["cssTheme"])
	}
	if got["accentColor"] != "red" || got["translucency"] != true {
		t.Fatalf("unexpected preserved fields: %v", got)
	}
}

func TestBuildPlanNoChange(t *testing.T) {
	root := t.TempDir()
	obsidianDir := filepath.Join(root, ".obsidian")
	if err := os.MkdirAll(obsidianDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(obsidianDir, "appearance.json"), []byte(`{"cssTheme":"Primary"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	v, err := vault.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := BuildPlan(v, "Primary")
	if err != nil {
		t.Fatal(err)
	}
	if plan.Changed {
		t.Fatal("expected no change")
	}
}

func TestApplyFontsUpdatesOnlyConfiguredFontFields(t *testing.T) {
	root := t.TempDir()
	obsidianDir := filepath.Join(root, ".obsidian")
	if err := os.MkdirAll(obsidianDir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(obsidianDir, "appearance.json")
	if err := os.WriteFile(path, []byte(`{"accentColor":"red","cssTheme":"Minimal","interfaceFontFamily":"Old UI","textFontFamily":"Old Text","monospaceFontFamily":"Old Mono"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	v, err := vault.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := BuildFontPlan(v, Fonts{
		Interface: "Maple Mono NF CN",
		Text:      "Maple Mono NF CN",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Changes) != 2 {
		t.Fatalf("got %d changes, want 2", len(plan.Changes))
	}
	if err := ApplyFonts(plan); err != nil {
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
	if got["interfaceFontFamily"] != "Maple Mono NF CN" {
		t.Fatalf("got interfaceFontFamily %v", got["interfaceFontFamily"])
	}
	if got["textFontFamily"] != "Maple Mono NF CN" {
		t.Fatalf("got textFontFamily %v", got["textFontFamily"])
	}
	if got["monospaceFontFamily"] != "Old Mono" {
		t.Fatalf("unexpected monospaceFontFamily %v", got["monospaceFontFamily"])
	}
	if got["accentColor"] != "red" || got["cssTheme"] != "Minimal" {
		t.Fatalf("unexpected preserved fields: %v", got)
	}
}

func TestBuildFontPlanNoChange(t *testing.T) {
	root := t.TempDir()
	obsidianDir := filepath.Join(root, ".obsidian")
	if err := os.MkdirAll(obsidianDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(obsidianDir, "appearance.json"), []byte(`{"interfaceFontFamily":"Maple Mono NF CN"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	v, err := vault.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := BuildFontPlan(v, Fonts{Interface: "Maple Mono NF CN"})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Changed() {
		t.Fatal("expected no change")
	}
}
