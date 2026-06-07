package sync

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"sort"

	"obsidian-preference-sync/internal/appearance"
	"obsidian-preference-sync/internal/appsettings"
	"obsidian-preference-sync/internal/config"
	gh "obsidian-preference-sync/internal/github"
	"obsidian-preference-sync/internal/install"
	"obsidian-preference-sync/internal/obsidiansettings"
	"obsidian-preference-sync/internal/registry"
	"obsidian-preference-sync/internal/settings"
	"obsidian-preference-sync/internal/theme"
	"obsidian-preference-sync/internal/vault"
	"obsidian-preference-sync/internal/vaultfiles"
)

type Options struct {
	VaultPath      string
	ConfigPath     string
	AllowDangerous bool
}

type Plan struct {
	VaultPath             string
	ConfigPath            string
	PluginInstalls        []install.Plan
	Themes                []theme.Plan
	ActiveTheme           *appearance.Plan
	VimMode               *appsettings.VimModePlan
	CommunityPluginsToAdd []string
	PluginSettings        []settings.CopyPlan
	Hotkeys               *obsidiansettings.CopyPlan
	VaultFiles            []vaultfiles.CopyPlan
	Warnings              []string
}

func (p Plan) Changed() bool {
	if len(p.PluginInstalls) > 0 || len(p.CommunityPluginsToAdd) > 0 {
		return true
	}
	for _, themePlan := range p.Themes {
		if themePlan.Changed() {
			return true
		}
	}
	if p.ActiveTheme != nil && p.ActiveTheme.Changed {
		return true
	}
	if p.VimMode != nil && p.VimMode.Changed {
		return true
	}
	for _, cp := range p.PluginSettings {
		if !cp.Skipped && len(cp.Files) > 0 {
			return true
		}
	}
	if p.Hotkeys != nil && p.Hotkeys.Changed {
		return true
	}
	for _, cp := range p.VaultFiles {
		if cp.Changed {
			return true
		}
	}
	return false
}

func BuildPlan(ctx context.Context, opts Options) (Plan, config.Config, vault.Vault, error) {
	cfg, err := config.Load(opts.ConfigPath)
	if err != nil {
		return Plan{}, config.Config{}, vault.Vault{}, err
	}
	if !opts.AllowDangerous {
		dangerous := config.DangerousSettingsIDs(cfg.PluginSettings)
		if len(dangerous) > 0 {
			sort.Strings(dangerous)
			return Plan{}, config.Config{}, vault.Vault{}, fmt.Errorf("dangerous plugin settings requested for %v; pass --allow-dangerous to allow", dangerous)
		}
	}
	v, err := vault.Open(opts.VaultPath)
	if err != nil {
		return Plan{}, config.Config{}, vault.Vault{}, err
	}

	plan := Plan{
		VaultPath:  v.Root,
		ConfigPath: cleanPath(opts.ConfigPath),
	}

	reg, err := registry.Fetch(ctx)
	if err != nil {
		return Plan{}, config.Config{}, vault.Vault{}, err
	}
	client := gh.NewClient()
	for _, pluginID := range cfg.Plugins {
		plugin, ok := reg.Lookup(pluginID)
		if !ok {
			return Plan{}, config.Config{}, vault.Vault{}, fmt.Errorf("plugin %q not found in Obsidian community registry", pluginID)
		}
		p := install.PlanPlugin(v, plugin)
		if p.NeedsInstall {
			plan.PluginInstalls = append(plan.PluginInstalls, p)
		}
	}
	if len(cfg.Themes) > 0 {
		themeReg, err := registry.FetchThemes(ctx)
		if err != nil {
			return Plan{}, config.Config{}, vault.Vault{}, err
		}
		for _, themeName := range cfg.Themes {
			themeEntry, ok := themeReg.Lookup(themeName)
			if !ok {
				return Plan{}, config.Config{}, vault.Vault{}, fmt.Errorf("theme %q not found in Obsidian community theme registry", themeName)
			}
			p, err := theme.BuildPlan(ctx, v, client, themeEntry)
			if err != nil {
				return Plan{}, config.Config{}, vault.Vault{}, err
			}
			plan.Themes = append(plan.Themes, p)
		}
	}
	if cfg.ActiveTheme != "" {
		p, err := appearance.BuildPlan(v, cfg.ActiveTheme)
		if err != nil {
			return Plan{}, config.Config{}, vault.Vault{}, err
		}
		plan.ActiveTheme = &p
	}
	if cfg.VimMode != nil {
		p, err := appsettings.BuildVimModePlan(v, *cfg.VimMode)
		if err != nil {
			return Plan{}, config.Config{}, vault.Vault{}, err
		}
		plan.VimMode = &p
	}

	added, _, err := v.UpsertEnabledPlugins(cfg.Plugins, true)
	if err != nil {
		return Plan{}, config.Config{}, vault.Vault{}, err
	}
	plan.CommunityPluginsToAdd = added

	for _, pluginID := range vault.SortedKeys(cfg.PluginSettings) {
		var cp settings.CopyPlan
		var err error
		if contains(cfg.Plugins, pluginID) {
			cp, err = settings.PlanAssumingInstalled(v, pluginID, cfg.PluginSettings[pluginID])
		} else {
			cp, err = settings.Plan(v, pluginID, cfg.PluginSettings[pluginID])
		}
		if err != nil {
			return Plan{}, config.Config{}, vault.Vault{}, err
		}
		if cp.Skipped {
			plan.Warnings = append(plan.Warnings, cp.Warning)
		}
		plan.PluginSettings = append(plan.PluginSettings, cp)
	}

	if cfg.Hotkeys != "" {
		cp, err := obsidiansettings.Plan(v, "hotkeys", cfg.Hotkeys)
		if err != nil {
			return Plan{}, config.Config{}, vault.Vault{}, err
		}
		plan.Hotkeys = &cp
	}
	for _, file := range cfg.VaultFiles {
		cp, err := vaultfiles.Plan(v, file)
		if err != nil {
			return Plan{}, config.Config{}, vault.Vault{}, err
		}
		plan.VaultFiles = append(plan.VaultFiles, cp)
	}

	return plan, cfg, v, nil
}

func Apply(ctx context.Context, plan Plan, cfg config.Config, v vault.Vault, verbose bool, stdout io.Writer) error {
	client := gh.NewClient()
	for _, p := range plan.PluginInstalls {
		plugin := registry.Plugin{ID: p.PluginID, Repo: p.Repo}
		fmt.Fprintf(stdout, "installing plugin %s from %s\n", p.PluginID, p.Repo)
		installed, err := install.InstallPlugin(ctx, v, client, plugin)
		if err != nil {
			return err
		}
		if verbose {
			fmt.Fprintf(stdout, "installed plugin %s files: %v\n", p.PluginID, installed.Files)
		}
	}

	for _, p := range plan.Themes {
		if !p.Changed() {
			continue
		}
		fmt.Fprintf(stdout, "theme %s: updating from %s\n", p.Name, p.Repo)
		if verbose {
			for _, file := range p.Files {
				if file.Changed {
					fmt.Fprintf(stdout, "theme %s: writing %s\n", p.Name, file.Name)
				}
			}
		}
		if err := theme.Apply(p); err != nil {
			return err
		}
	}
	if plan.ActiveTheme != nil && plan.ActiveTheme.Changed {
		fmt.Fprintf(stdout, "active theme: setting to %s\n", plan.ActiveTheme.ThemeName)
		if err := appearance.Apply(*plan.ActiveTheme); err != nil {
			return err
		}
	}
	if plan.VimMode != nil && plan.VimMode.Changed {
		fmt.Fprintf(stdout, "vim mode: setting to %t\n", plan.VimMode.Enable)
		if err := appsettings.ApplyVimMode(*plan.VimMode); err != nil {
			return err
		}
	}

	added, enabledChanged, err := v.UpsertEnabledPlugins(cfg.Plugins, false)
	if err != nil {
		return err
	}
	if enabledChanged {
		fmt.Fprintf(stdout, "community-plugins.json: added %v\n", added)
	}

	for _, cp := range plan.PluginSettings {
		if cp.Skipped || len(cp.Files) == 0 {
			continue
		}
		fmt.Fprintf(stdout, "settings %s: copying %d file(s)\n", cp.PluginID, len(cp.Files))
		if verbose {
			for _, file := range cp.Files {
				fmt.Fprintf(stdout, "settings %s: copying %s\n", cp.PluginID, file)
			}
		}
		if err := settings.Apply(cp, false); err != nil {
			return err
		}
	}

	if plan.Hotkeys != nil && plan.Hotkeys.Changed {
		fmt.Fprintf(stdout, "hotkeys: copying to %s\n", plan.Hotkeys.Target)
		if verbose {
			fmt.Fprintf(stdout, "hotkeys: source %s\n", plan.Hotkeys.Source)
		}
		if err := obsidiansettings.Apply(*plan.Hotkeys, false); err != nil {
			return err
		}
	}
	for _, cp := range plan.VaultFiles {
		if !cp.Changed {
			continue
		}
		fmt.Fprintf(stdout, "vault file: copying to %s\n", cp.Target)
		if verbose {
			fmt.Fprintf(stdout, "vault file: source %s\n", cp.Source)
		}
		if err := vaultfiles.Apply(cp, false); err != nil {
			return err
		}
	}
	return nil
}

func RenderPlan(plan Plan, verbose bool, stdout io.Writer, stderr io.Writer) {
	if verbose {
		fmt.Fprintf(stdout, "vault: %s\n", plan.VaultPath)
		fmt.Fprintf(stdout, "config: %s\n", plan.ConfigPath)
	}

	for _, warning := range plan.Warnings {
		fmt.Fprintf(stderr, "warning: %s\n", warning)
	}
	for _, p := range plan.PluginInstalls {
		fmt.Fprintf(stdout, "plugin %s: will install from %s\n", p.PluginID, p.Repo)
	}
	for _, p := range plan.Themes {
		if !p.Changed() {
			if verbose {
				fmt.Fprintf(stdout, "theme %s: no changes\n", p.Name)
			}
			continue
		}
		var files []string
		for _, file := range p.Files {
			if file.Changed {
				files = append(files, file.Name)
			}
		}
		fmt.Fprintf(stdout, "theme %s: will update %v from %s\n", p.Name, files, p.Repo)
	}
	if plan.ActiveTheme != nil {
		if plan.ActiveTheme.Changed {
			fmt.Fprintf(stdout, "active theme: will set to %s\n", plan.ActiveTheme.ThemeName)
		} else if verbose {
			fmt.Fprintf(stdout, "active theme: already %s\n", plan.ActiveTheme.ThemeName)
		}
	}
	if plan.VimMode != nil {
		if plan.VimMode.Changed {
			fmt.Fprintf(stdout, "vim mode: will set to %t\n", plan.VimMode.Enable)
		} else if verbose {
			fmt.Fprintf(stdout, "vim mode: already %t\n", plan.VimMode.Enable)
		}
	}
	if len(plan.CommunityPluginsToAdd) > 0 {
		fmt.Fprintf(stdout, "community-plugins.json: will add %v\n", plan.CommunityPluginsToAdd)
	} else if verbose {
		fmt.Fprintln(stdout, "community-plugins.json: no missing plugin ids")
	}
	for _, cp := range plan.PluginSettings {
		if cp.Skipped {
			continue
		}
		if len(cp.Files) == 0 {
			if verbose {
				fmt.Fprintf(stdout, "settings %s: no changes\n", cp.PluginID)
			}
			continue
		}
		fmt.Fprintf(stdout, "settings %s: will copy %d file(s) from %s\n", cp.PluginID, len(cp.Files), cp.Source)
	}
	if plan.Hotkeys != nil {
		if plan.Hotkeys.Changed {
			fmt.Fprintf(stdout, "hotkeys: will copy %s to %s\n", plan.Hotkeys.Source, plan.Hotkeys.Target)
		} else if verbose {
			fmt.Fprintln(stdout, "hotkeys: no changes")
		}
	}
	for _, cp := range plan.VaultFiles {
		if cp.Changed {
			fmt.Fprintf(stdout, "vault file: will copy %s to %s\n", cp.Source, cp.Target)
		} else if verbose {
			fmt.Fprintf(stdout, "vault file: no changes for %s\n", cp.Target)
		}
	}
	if !plan.Changed() {
		fmt.Fprintln(stdout, "no changes")
	}
}

func cleanPath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
