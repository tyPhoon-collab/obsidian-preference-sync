package theme

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPlanChanged(t *testing.T) {
	plan := Plan{
		Files: []FilePlan{
			{Name: "manifest.json", Changed: false},
			{Name: "theme.css", Changed: true},
		},
	}
	if !plan.Changed() {
		t.Fatal("expected changed plan")
	}
}

func TestApplyWritesOnlyChangedFiles(t *testing.T) {
	dir := t.TempDir()
	plan := Plan{
		Name: "Primary",
		Dir:  dir,
		Files: []FilePlan{
			{Name: "manifest.json", Data: []byte("new manifest"), Changed: false},
			{Name: "theme.css", Data: []byte("new css"), Changed: true},
		},
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte("old manifest"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Apply(plan); err != nil {
		t.Fatal(err)
	}
	manifest, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(manifest) != "old manifest" {
		t.Fatalf("unchanged file was overwritten: %q", manifest)
	}
	css, err := os.ReadFile(filepath.Join(dir, "theme.css"))
	if err != nil {
		t.Fatal(err)
	}
	if string(css) != "new css" {
		t.Fatalf("got %q", css)
	}
}
