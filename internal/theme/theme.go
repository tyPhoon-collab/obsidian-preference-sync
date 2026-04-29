package theme

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"obsidian-preference-sync/internal/fileutil"
	gh "obsidian-preference-sync/internal/github"
	"obsidian-preference-sync/internal/registry"
	"obsidian-preference-sync/internal/vault"
)

var requiredFiles = []string{"manifest.json", "theme.css"}

type FilePlan struct {
	Name    string
	Data    []byte
	Changed bool
}

type Plan struct {
	Name  string
	Repo  string
	Dir   string
	Files []FilePlan
}

func BuildPlan(ctx context.Context, v vault.Vault, client gh.Client, theme registry.Theme) (Plan, error) {
	plan := Plan{
		Name: theme.Name,
		Repo: theme.Repo,
		Dir:  filepath.Join(v.ObsidianDir, "themes", theme.Name),
	}
	for _, name := range requiredFiles {
		data, err := client.DownloadRepoFile(ctx, theme.Repo, name)
		if err != nil {
			return plan, err
		}
		target := filepath.Join(plan.Dir, name)
		same, err := fileutil.FileContentEqualBytes(data, target)
		if err != nil {
			return plan, err
		}
		plan.Files = append(plan.Files, FilePlan{
			Name:    name,
			Data:    data,
			Changed: !same,
		})
	}
	return plan, nil
}

func (p Plan) Changed() bool {
	for _, file := range p.Files {
		if file.Changed {
			return true
		}
	}
	return false
}

func Apply(p Plan) error {
	if !p.Changed() {
		return nil
	}
	if err := os.MkdirAll(p.Dir, 0o755); err != nil {
		return fmt.Errorf("create theme directory %s: %w", p.Dir, err)
	}
	for _, file := range p.Files {
		if !file.Changed {
			continue
		}
		if err := fileutil.WriteFileAtomic(filepath.Join(p.Dir, file.Name), file.Data, 0o644); err != nil {
			return err
		}
	}
	return nil
}
