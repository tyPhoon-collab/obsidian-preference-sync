package install

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

var requiredAssets = []string{"manifest.json", "main.js"}
var optionalAssets = []string{"styles.css"}

type Plan struct {
	PluginID     string
	Repo         string
	Installed    bool
	NeedsInstall bool
	Files        []string
}

func PlanPlugin(v vault.Vault, plugin registry.Plugin) Plan {
	installed := pluginHasRequiredAssets(v.PluginDir(plugin.ID))
	return Plan{
		PluginID:     plugin.ID,
		Repo:         plugin.Repo,
		Installed:    installed,
		NeedsInstall: !installed,
	}
}

func InstallPlugin(ctx context.Context, v vault.Vault, client gh.Client, plugin registry.Plugin) (Plan, error) {
	plan := PlanPlugin(v, plugin)
	if !plan.NeedsInstall {
		return plan, nil
	}

	release, err := client.LatestRelease(ctx, plugin.Repo)
	if err != nil {
		return plan, err
	}
	assetURLs := map[string]string{}
	for _, name := range requiredAssets {
		url, ok := release.AssetURL(name)
		if !ok {
			return plan, fmt.Errorf("release %s for %s does not contain required asset %s", release.TagName, plugin.ID, name)
		}
		assetURLs[name] = url
	}
	for _, name := range optionalAssets {
		if url, ok := release.AssetURL(name); ok {
			assetURLs[name] = url
		}
	}

	pluginDir := v.PluginDir(plugin.ID)
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		return plan, fmt.Errorf("create plugin directory %s: %w", pluginDir, err)
	}
	for name, url := range assetURLs {
		data, err := client.Download(ctx, url)
		if err != nil {
			return plan, err
		}
		if err := fileutil.WriteFileAtomic(filepath.Join(pluginDir, name), data, 0o644); err != nil {
			return plan, err
		}
		plan.Files = append(plan.Files, name)
	}
	return plan, nil
}

func pluginHasRequiredAssets(pluginDir string) bool {
	for _, name := range requiredAssets {
		info, err := os.Stat(filepath.Join(pluginDir, name))
		if err != nil || info.IsDir() {
			return false
		}
	}
	return true
}
