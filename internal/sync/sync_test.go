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

	if !strings.Contains(stdout.String(), "vault file: will copy /tmp/source.vimrc to /tmp/vault/.obsidian.vimrc") {
		t.Fatalf("unexpected output: %q", stdout.String())
	}
}
