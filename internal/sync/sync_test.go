package sync

import (
	"bytes"
	"strings"
	"testing"

	"obsidian-preference-sync/internal/appearance"
	"obsidian-preference-sync/internal/obsidiansettings"
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
