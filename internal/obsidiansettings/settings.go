package obsidiansettings

import (
	"fmt"
	"os"
	"path/filepath"

	"obsidian-preference-sync/internal/fileutil"
	"obsidian-preference-sync/internal/vault"
)

var targetFiles = map[string]string{
	"hotkeys": "hotkeys.json",
}

type CopyPlan struct {
	Name    string
	Source  string
	Target  string
	Changed bool
}

func Plan(v vault.Vault, name string, source string) (CopyPlan, error) {
	targetFile, ok := targetFiles[name]
	if !ok {
		return CopyPlan{}, fmt.Errorf("unsupported obsidian setting %q", name)
	}
	info, err := os.Stat(source)
	if err != nil {
		return CopyPlan{}, fmt.Errorf("obsidian setting source for %s does not exist: %s", name, source)
	}
	if info.IsDir() {
		return CopyPlan{}, fmt.Errorf("obsidian setting source for %s is a directory: %s", name, source)
	}
	target := filepath.Join(v.ObsidianDir, targetFile)
	same, err := fileutil.FileContentEqual(source, target)
	if err != nil {
		return CopyPlan{}, err
	}
	return CopyPlan{
		Name:    name,
		Source:  source,
		Target:  target,
		Changed: !same,
	}, nil
}

func Apply(cp CopyPlan, dryRun bool) error {
	if dryRun || !cp.Changed {
		return nil
	}
	info, err := os.Stat(cp.Source)
	if err != nil {
		return fmt.Errorf("stat obsidian setting source %s: %w", cp.Source, err)
	}
	return fileutil.CopyFileAtomic(cp.Source, cp.Target, info.Mode().Perm())
}
