package sync

import (
	"bytes"
	"strings"
	"testing"

	"obsidian-preference-sync/internal/obsidiansettings"
)

func TestPlanChangedIncludesVimrc(t *testing.T) {
	plan := Plan{
		Vimrc: &obsidiansettings.CopyPlan{
			Name:    "vimrc",
			Source:  "/tmp/source.vimrc",
			Target:  "/tmp/vault/.obsidian.vimrc",
			Changed: true,
		},
	}
	if !plan.Changed() {
		t.Fatal("expected changed plan")
	}
}

func TestRenderPlanIncludesVimrc(t *testing.T) {
	var stdout bytes.Buffer
	RenderPlan(Plan{
		Vimrc: &obsidiansettings.CopyPlan{
			Name:    "vimrc",
			Source:  "/tmp/source.vimrc",
			Target:  "/tmp/vault/.obsidian.vimrc",
			Changed: true,
		},
	}, false, &stdout, &bytes.Buffer{})

	if !strings.Contains(stdout.String(), "vimrc: will copy /tmp/source.vimrc to /tmp/vault/.obsidian.vimrc") {
		t.Fatalf("unexpected output: %q", stdout.String())
	}
}
