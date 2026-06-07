package sync

import (
	"bytes"
	"strings"
	"testing"

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
