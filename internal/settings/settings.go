package settings

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"obsidian-preference-sync/internal/fileutil"
	"obsidian-preference-sync/internal/vault"
)

var excludedNames = map[string]bool{
	"main.js":       true,
	"manifest.json": true,
	"styles.css":    true,
	"node_modules":  true,
}

type CopyPlan struct {
	PluginID string
	Source   string
	Target   string
	Files    []string
	Skipped  bool
	Warning  string
}

func Plan(v vault.Vault, pluginID string, source string) (CopyPlan, error) {
	return plan(v, pluginID, source, false)
}

func PlanAssumingInstalled(v vault.Vault, pluginID string, source string) (CopyPlan, error) {
	return plan(v, pluginID, source, true)
}

func plan(v vault.Vault, pluginID string, source string, assumeInstalled bool) (CopyPlan, error) {
	cp := CopyPlan{
		PluginID: pluginID,
		Source:   source,
		Target:   v.PluginDir(pluginID),
	}
	if !assumeInstalled && !v.IsPluginInstalled(pluginID) {
		cp.Skipped = true
		cp.Warning = fmt.Sprintf("plugin %q is not installed; skipping settings", pluginID)
		return cp, nil
	}
	info, err := os.Stat(source)
	if err != nil {
		return cp, fmt.Errorf("settings source for %s does not exist: %s", pluginID, source)
	}
	if !info.IsDir() {
		return cp, fmt.Errorf("settings source for %s is not a directory: %s", pluginID, source)
	}
	err = filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == source {
			return nil
		}
		name := entry.Name()
		if excludedNames[name] {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		target := filepath.Join(cp.Target, rel)
		same, err := fileutil.FileContentEqual(path, target)
		if err != nil {
			return err
		}
		if !same {
			cp.Files = append(cp.Files, rel)
		}
		return nil
	})
	if err != nil {
		return cp, fmt.Errorf("scan settings source for %s: %w", pluginID, err)
	}
	return cp, nil
}

func Apply(cp CopyPlan, dryRun bool) error {
	if cp.Skipped || dryRun {
		return nil
	}
	for _, rel := range cp.Files {
		src := filepath.Join(cp.Source, rel)
		dst := filepath.Join(cp.Target, rel)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return fmt.Errorf("create settings directory %s: %w", filepath.Dir(dst), err)
		}
		info, err := os.Stat(src)
		if err != nil {
			return fmt.Errorf("stat settings file %s: %w", src, err)
		}
		if err := fileutil.CopyFileAtomic(src, dst, info.Mode().Perm()); err != nil {
			return err
		}
	}
	return nil
}
