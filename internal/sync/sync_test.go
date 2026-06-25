package sync

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"obsidian-preference-sync/internal/appearance"
	"obsidian-preference-sync/internal/config"
	"obsidian-preference-sync/internal/obsidiansettings"
	"obsidian-preference-sync/internal/vault"
	"obsidian-preference-sync/internal/vaultfiles"
)

func TestPlanChangedIncludesVaultFiles(t *testing.T) {
	plan := Plan{
		VaultFiles: []vaultfiles.CopyPlan{
			{
				Source:  "/tmp/source.vimrc",
				Target:  "/tmp/vault/.obsidian.vimrc",
				Changed: true,
			},
		},
	}
	if !plan.Changed() {
		t.Fatal("expected changed plan")
	}
}

func TestRenderPlanIncludesVaultFiles(t *testing.T) {
	var stdout bytes.Buffer
	RenderPlan(Plan{
		VaultFiles: []vaultfiles.CopyPlan{
			{
				Source:  "/tmp/source.vimrc",
				Target:  "/tmp/vault/.obsidian.vimrc",
				Changed: true,
			},
		},
	}, false, &stdout, &bytes.Buffer{})

	if !strings.Contains(stdout.String(), "Plan: 1 change") {
		t.Fatalf("unexpected output: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Files To Copy") {
		t.Fatalf("unexpected output: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "~ vault-file") {
		t.Fatalf("unexpected output: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "/tmp/vault/.obsidian.vimrc") {
		t.Fatalf("unexpected output: %q", stdout.String())
	}
}

func TestRenderPlanIncludesFonts(t *testing.T) {
	var stdout bytes.Buffer
	RenderPlan(Plan{
		Fonts: &appearance.FontPlan{
			Fonts: appearance.Fonts{Text: "Maple Mono NF CN"},
			Changes: []appearance.FontChange{
				{Name: "text-font", Key: "textFontFamily", Value: "Maple Mono NF CN"},
			},
		},
	}, false, &stdout, &bytes.Buffer{})

	if !strings.Contains(stdout.String(), "Plan: 1 change") {
		t.Fatalf("unexpected output: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Obsidian Settings") {
		t.Fatalf("unexpected output: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "~ text-font") {
		t.Fatalf("unexpected output: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Maple Mono NF CN") {
		t.Fatalf("unexpected output: %q", stdout.String())
	}
}

func TestRenderPlanIncludesObsidianSettings(t *testing.T) {
	var stdout bytes.Buffer
	RenderPlan(Plan{
		VaultPath: "/tmp/vault",
		ObsidianSettings: []obsidiansettings.CopyPlan{
			{
				Name:    "command-palette",
				Source:  "/tmp/source/command-palette.json",
				Target:  "/tmp/vault/.obsidian/command-palette.json",
				Changed: true,
			},
		},
	}, false, &stdout, &bytes.Buffer{})

	if !strings.Contains(stdout.String(), "Plan: 1 change") {
		t.Fatalf("unexpected output: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Files To Copy") {
		t.Fatalf("unexpected output: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "~ command-palette") {
		t.Fatalf("unexpected output: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), ".obsidian/command-palette.json") {
		t.Fatalf("unexpected output: %q", stdout.String())
	}
}

func TestUnmanagedEnabledPlugins(t *testing.T) {
	root := t.TempDir()
	obsidianDir := filepath.Join(root, ".obsidian")
	if err := os.MkdirAll(obsidianDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(obsidianDir, "community-plugins.json"), []byte(`["open-tab-settings","obsidian-linter","disable-tabs"]`), 0o644); err != nil {
		t.Fatal(err)
	}
	v, err := vault.Open(root)
	if err != nil {
		t.Fatal(err)
	}

	got, err := unmanagedEnabledPlugins(v, []string{"obsidian-linter", "disable-tabs"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != "open-tab-settings" {
		t.Fatalf("got %v, want [open-tab-settings]", got)
	}
}

func TestRenderPlanPrintsWarnings(t *testing.T) {
	var stderr bytes.Buffer
	RenderPlan(Plan{
		Warnings: []string{`enabled plugin "open-tab-settings" is not listed in config plugins; disable it in Obsidian or add it to config`},
	}, false, &bytes.Buffer{}, &stderr)

	if !strings.Contains(stderr.String(), `warning: enabled plugin "open-tab-settings" is not listed in config plugins`) {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestApplyPrintsWarnings(t *testing.T) {
	var stdout bytes.Buffer
	err := Apply(context.Background(), Plan{
		Warnings: []string{`enabled plugin "open-tab-settings" is not listed in config plugins; disable it in Obsidian or add it to config`},
	}, config.Config{}, vault.Vault{}, false, &stdout)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(stdout.String(), `warning: enabled plugin "open-tab-settings" is not listed in config plugins`) {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
}
