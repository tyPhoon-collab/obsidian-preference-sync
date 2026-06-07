package vaultfiles

import (
	"os"
	"path/filepath"
	"testing"

	"obsidian-preference-sync/internal/config"
	"obsidian-preference-sync/internal/vault"
)

func TestApplyCopiesVaultFileToVaultRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".obsidian"), 0o755); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "source.vimrc")
	if err := os.WriteFile(source, []byte("set clipboard=unnamed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	v, err := vault.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	cp, err := Plan(v, config.FileCopy{Source: source, Target: ".obsidian.vimrc"})
	if err != nil {
		t.Fatal(err)
	}
	if cp.Target != filepath.Join(root, ".obsidian.vimrc") {
		t.Fatalf("got target %q", cp.Target)
	}
	if err := Apply(cp, false); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(root, ".obsidian.vimrc"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "set clipboard=unnamed\n" {
		t.Fatalf("got %q", got)
	}
}

func TestApplyCreatesNestedVaultFileDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".obsidian"), 0o755); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "source.json")
	if err := os.WriteFile(source, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	v, err := vault.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	cp, err := Plan(v, config.FileCopy{Source: source, Target: "settings/source.json"})
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(cp, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "settings", "source.json")); err != nil {
		t.Fatal(err)
	}
}

func TestPlanMarksUnchangedVaultFile(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".obsidian"), 0o755); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "spacekeys.yml")
	content := []byte("items: {}\n")
	if err := os.WriteFile(source, content, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "target-spacekeys.yml"), content, 0o644); err != nil {
		t.Fatal(err)
	}
	v, err := vault.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	cp, err := Plan(v, config.FileCopy{Source: source, Target: "target-spacekeys.yml"})
	if err != nil {
		t.Fatal(err)
	}
	if cp.Changed {
		t.Fatal("expected unchanged vault file")
	}
}
