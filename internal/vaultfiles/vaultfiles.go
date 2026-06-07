package vaultfiles

import (
	"fmt"
	"os"
	"path/filepath"

	"obsidian-preference-sync/internal/config"
	"obsidian-preference-sync/internal/fileutil"
	"obsidian-preference-sync/internal/vault"
)

type CopyPlan struct {
	Source  string
	Target  string
	Changed bool
}

func Plan(v vault.Vault, file config.FileCopy) (CopyPlan, error) {
	info, err := os.Stat(file.Source)
	if err != nil {
		return CopyPlan{}, fmt.Errorf("vault file source does not exist: %s", file.Source)
	}
	if info.IsDir() {
		return CopyPlan{}, fmt.Errorf("vault file source is a directory: %s", file.Source)
	}
	target := filepath.Join(v.Root, file.Target)
	same, err := fileutil.FileContentEqual(file.Source, target)
	if err != nil {
		return CopyPlan{}, err
	}
	return CopyPlan{
		Source:  file.Source,
		Target:  target,
		Changed: !same,
	}, nil
}

func Apply(cp CopyPlan, dryRun bool) error {
	if dryRun || !cp.Changed {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(cp.Target), 0o755); err != nil {
		return fmt.Errorf("create vault file directory %s: %w", filepath.Dir(cp.Target), err)
	}
	info, err := os.Stat(cp.Source)
	if err != nil {
		return fmt.Errorf("stat vault file source %s: %w", cp.Source, err)
	}
	return fileutil.CopyFileAtomic(cp.Source, cp.Target, info.Mode().Perm())
}
