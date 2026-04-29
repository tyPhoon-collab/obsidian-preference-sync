package vault

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"obsidian-preference-sync/internal/fileutil"
)

type Vault struct {
	Root        string
	ObsidianDir string
	PluginsDir  string
}

func Open(path string) (Vault, error) {
	root, err := filepath.Abs(path)
	if err != nil {
		return Vault{}, err
	}
	info, err := os.Stat(root)
	if err != nil {
		return Vault{}, fmt.Errorf("vault does not exist: %s", root)
	}
	if !info.IsDir() {
		return Vault{}, fmt.Errorf("vault is not a directory: %s", root)
	}
	obsidianDir := filepath.Join(root, ".obsidian")
	info, err = os.Stat(obsidianDir)
	if err != nil {
		return Vault{}, fmt.Errorf(".obsidian directory does not exist in vault: %s", obsidianDir)
	}
	if !info.IsDir() {
		return Vault{}, fmt.Errorf(".obsidian is not a directory: %s", obsidianDir)
	}
	return Vault{
		Root:        root,
		ObsidianDir: obsidianDir,
		PluginsDir:  filepath.Join(obsidianDir, "plugins"),
	}, nil
}

func (v Vault) PluginDir(pluginID string) string {
	return filepath.Join(v.PluginsDir, pluginID)
}

func (v Vault) IsPluginInstalled(pluginID string) bool {
	info, err := os.Stat(v.PluginDir(pluginID))
	return err == nil && info.IsDir()
}

func (v Vault) ReadEnabledPlugins() ([]string, error) {
	path := filepath.Join(v.ObsidianDir, "community-plugins.json")
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return []string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read community plugins: %w", err)
	}
	if len(data) == 0 {
		return []string{}, nil
	}
	var plugins []string
	if err := json.Unmarshal(data, &plugins); err != nil {
		return nil, fmt.Errorf("parse community-plugins.json: %w", err)
	}
	return plugins, nil
}

func (v Vault) UpsertEnabledPlugins(pluginIDs []string, dryRun bool) ([]string, bool, error) {
	existing, err := v.ReadEnabledPlugins()
	if err != nil {
		return nil, false, err
	}
	seen := map[string]bool{}
	for _, id := range existing {
		seen[id] = true
	}
	var added []string
	for _, id := range pluginIDs {
		if !seen[id] {
			existing = append(existing, id)
			seen[id] = true
			added = append(added, id)
		}
	}
	if len(added) == 0 {
		return added, false, nil
	}
	if dryRun {
		return added, true, nil
	}
	if err := os.MkdirAll(v.PluginsDir, 0o755); err != nil {
		return nil, false, fmt.Errorf("create plugins directory: %w", err)
	}
	path := filepath.Join(v.ObsidianDir, "community-plugins.json")
	if err := fileutil.WriteJSONAtomic(path, existing, 0o644); err != nil {
		return nil, false, err
	}
	return added, true, nil
}

func SortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
