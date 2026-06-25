package sync

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

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
	Fonts                 *appearance.FontPlan
	VimMode               *appsettings.VimModePlan
	ShowLineNumber        *appsettings.ShowLineNumberPlan
	CommunityPluginsToAdd []string
	PluginSettings        []settings.CopyPlan
	Hotkeys               *obsidiansettings.CopyPlan
	ObsidianSettings      []obsidiansettings.CopyPlan
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
	if p.Fonts != nil && p.Fonts.Changed() {
		return true
	}
	if p.VimMode != nil && p.VimMode.Changed {
		return true
	}
	if p.ShowLineNumber != nil && p.ShowLineNumber.Changed {
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
	for _, cp := range p.ObsidianSettings {
		if cp.Changed {
			return true
		}
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
	if !cfg.Fonts.Empty() {
		p, err := appearance.BuildFontPlan(v, appearance.Fonts{
			Interface: cfg.Fonts.Interface,
			Text:      cfg.Fonts.Text,
			Monospace: cfg.Fonts.Monospace,
		})
		if err != nil {
			return Plan{}, config.Config{}, vault.Vault{}, err
		}
		plan.Fonts = &p
	}
	if cfg.VimMode != nil {
		p, err := appsettings.BuildVimModePlan(v, *cfg.VimMode)
		if err != nil {
			return Plan{}, config.Config{}, vault.Vault{}, err
		}
		plan.VimMode = &p
	}
	if cfg.ShowLineNumber != nil {
		p, err := appsettings.BuildShowLineNumberPlan(v, *cfg.ShowLineNumber)
		if err != nil {
			return Plan{}, config.Config{}, vault.Vault{}, err
		}
		plan.ShowLineNumber = &p
	}
	unmanaged, err := unmanagedEnabledPlugins(v, cfg.Plugins)
	if err != nil {
		return Plan{}, config.Config{}, vault.Vault{}, err
	}
	for _, pluginID := range unmanaged {
		plan.Warnings = append(plan.Warnings, fmt.Sprintf("enabled plugin %q is not listed in config plugins; disable it in Obsidian or add it to config", pluginID))
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
	for _, setting := range []struct {
		name   string
		source string
	}{
		{name: "command-palette", source: cfg.CommandPalette},
	} {
		if setting.source == "" {
			continue
		}
		cp, err := obsidiansettings.Plan(v, setting.name, setting.source)
		if err != nil {
			return Plan{}, config.Config{}, vault.Vault{}, err
		}
		plan.ObsidianSettings = append(plan.ObsidianSettings, cp)
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
	out := newStyle(stdout)
	fmt.Fprintf(stdout, "%s\n", out.heading(applySummary(plan)))
	printWarnings(plan, stdout)
	if !plan.Changed() {
		return nil
	}

	client := gh.NewClient()
	printedPlugins := false
	for _, p := range plan.PluginInstalls {
		plugin := registry.Plugin{ID: p.PluginID, Repo: p.Repo}
		printSectionHeader(stdout, &printedPlugins, "Plugins")
		action := "install"
		if contains(plan.CommunityPluginsToAdd, p.PluginID) {
			action = "install+enable"
		}
		fmt.Fprintf(stdout, "  %s %-14s %-30s %s\n", out.add("+"), action, p.PluginID, p.Repo)
		installed, err := install.InstallPlugin(ctx, v, client, plugin)
		if err != nil {
			return err
		}
		if verbose {
			fmt.Fprintf(stdout, "    files %s\n", strings.Join(installed.Files, ", "))
		}
	}

	printedThemes := false
	for _, p := range plan.Themes {
		if !p.Changed() {
			continue
		}
		var files []string
		for _, file := range p.Files {
			if file.Changed {
				files = append(files, file.Name)
			}
		}
		printSectionHeader(stdout, &printedThemes, "Themes")
		fmt.Fprintf(stdout, "  %s %-14s %-30s %s\n", out.change("~"), "update", p.Name, strings.Join(files, ", "))
		if verbose {
			fmt.Fprintf(stdout, "    source %s\n", p.Repo)
		}
		if err := theme.Apply(p); err != nil {
			return err
		}
	}

	printedObsidian := false
	if plan.ActiveTheme != nil && plan.ActiveTheme.Changed {
		printSectionHeader(stdout, &printedObsidian, "Obsidian Settings")
		fmt.Fprintf(stdout, "  %s %-14s %s\n", out.change("~"), "active-theme", plan.ActiveTheme.ThemeName)
		if err := appearance.Apply(*plan.ActiveTheme); err != nil {
			return err
		}
	}
	if plan.Fonts != nil && plan.Fonts.Changed() {
		printSectionHeader(stdout, &printedObsidian, "Obsidian Settings")
		for _, change := range plan.Fonts.Changes {
			fmt.Fprintf(stdout, "  %s %-14s %s\n", out.change("~"), change.Name, change.Value)
		}
		if err := appearance.ApplyFonts(*plan.Fonts); err != nil {
			return err
		}
	}
	if plan.VimMode != nil && plan.VimMode.Changed {
		printSectionHeader(stdout, &printedObsidian, "Obsidian Settings")
		fmt.Fprintf(stdout, "  %s %-14s %t\n", out.change("~"), "vim-mode", plan.VimMode.Enable)
		if err := appsettings.ApplyVimMode(*plan.VimMode); err != nil {
			return err
		}
	}
	if plan.ShowLineNumber != nil && plan.ShowLineNumber.Changed {
		printSectionHeader(stdout, &printedObsidian, "Obsidian Settings")
		fmt.Fprintf(stdout, "  %s %-14s %t\n", out.change("~"), "line-numbers", plan.ShowLineNumber.Enable)
		if err := appsettings.ApplyShowLineNumber(*plan.ShowLineNumber); err != nil {
			return err
		}
	}

	for _, pluginID := range plan.CommunityPluginsToAdd {
		if installPlanned(plan.PluginInstalls, pluginID) {
			continue
		}
		printSectionHeader(stdout, &printedPlugins, "Plugins")
		fmt.Fprintf(stdout, "  %s %-14s %s\n", out.add("+"), "enable", pluginID)
	}
	added, enabledChanged, err := v.UpsertEnabledPlugins(cfg.Plugins, false)
	if err != nil {
		return err
	}
	if verbose && enabledChanged && len(added) > 0 {
		fmt.Fprintf(stdout, "    community-plugins.json added %s\n", strings.Join(added, ", "))
	}

	printedFiles := false
	for _, cp := range plan.PluginSettings {
		if cp.Skipped || len(cp.Files) == 0 {
			continue
		}
		printSectionHeader(stdout, &printedFiles, "Files To Copy")
		fmt.Fprintf(stdout, "  %s %-24s %s\n", out.change("~"), "plugin/"+cp.PluginID, fileCount(len(cp.Files)))
		if verbose {
			for _, file := range cp.Files {
				fmt.Fprintf(stdout, "    file %s\n", file)
			}
			fmt.Fprintf(stdout, "    source %s\n", cp.Source)
		}
		if err := settings.Apply(cp, false); err != nil {
			return err
		}
	}

	if plan.Hotkeys != nil && plan.Hotkeys.Changed {
		printSectionHeader(stdout, &printedFiles, "Files To Copy")
		fmt.Fprintf(stdout, "  %s %-24s %s\n", out.change("~"), "hotkeys", relativeTarget(plan, plan.Hotkeys.Target))
		if verbose {
			fmt.Fprintf(stdout, "    source %s\n", plan.Hotkeys.Source)
		}
		if err := obsidiansettings.Apply(*plan.Hotkeys, false); err != nil {
			return err
		}
	}
	for _, cp := range plan.ObsidianSettings {
		if !cp.Changed {
			continue
		}
		printSectionHeader(stdout, &printedFiles, "Files To Copy")
		fmt.Fprintf(stdout, "  %s %-24s %s\n", out.change("~"), cp.Name, relativeTarget(plan, cp.Target))
		if verbose {
			fmt.Fprintf(stdout, "    source %s\n", cp.Source)
		}
		if err := obsidiansettings.Apply(cp, false); err != nil {
			return err
		}
	}
	for _, cp := range plan.VaultFiles {
		if !cp.Changed {
			continue
		}
		printSectionHeader(stdout, &printedFiles, "Files To Copy")
		fmt.Fprintf(stdout, "  %s %-24s %s\n", out.change("~"), "vault-file", relativeTarget(plan, cp.Target))
		if verbose {
			fmt.Fprintf(stdout, "    source %s\n", cp.Source)
		}
		if err := vaultfiles.Apply(cp, false); err != nil {
			return err
		}
	}

	fmt.Fprintln(stdout, "\nDone")
	fmt.Fprintln(stdout, "  Restart Obsidian to ensure plugin and setting changes are fully applied.")
	return nil
}

func RenderPlan(plan Plan, verbose bool, stdout io.Writer, stderr io.Writer) {
	out := newStyle(stdout)
	errStyle := newStyle(stderr)

	fmt.Fprintf(stdout, "%s\n", out.heading(planSummary(plan)))
	if verbose {
		fmt.Fprintf(stdout, "\nVault  %s\n", plan.VaultPath)
		fmt.Fprintf(stdout, "Config %s\n", plan.ConfigPath)
	}

	printWarningsWithStyle(plan, stderr, errStyle)

	printedPlugins := false
	for _, p := range plan.PluginInstalls {
		printSectionHeader(stdout, &printedPlugins, "Plugins")
		action := "install"
		if contains(plan.CommunityPluginsToAdd, p.PluginID) {
			action = "install+enable"
		}
		fmt.Fprintf(stdout, "  %s %-14s %-30s %s\n", out.add("+"), action, p.PluginID, p.Repo)
	}
	for _, pluginID := range plan.CommunityPluginsToAdd {
		if installPlanned(plan.PluginInstalls, pluginID) {
			continue
		}
		printSectionHeader(stdout, &printedPlugins, "Plugins")
		fmt.Fprintf(stdout, "  %s %-14s %s\n", out.add("+"), "enable", pluginID)
	}

	printedThemes := false
	for _, p := range plan.Themes {
		if !p.Changed() {
			continue
		}
		var files []string
		for _, file := range p.Files {
			if file.Changed {
				files = append(files, file.Name)
			}
		}
		printSectionHeader(stdout, &printedThemes, "Themes")
		fmt.Fprintf(stdout, "  %s %-14s %-30s %s\n", out.change("~"), "update", p.Name, strings.Join(files, ", "))
		if verbose {
			fmt.Fprintf(stdout, "    source %s\n", p.Repo)
		}
	}

	printedObsidian := false
	if plan.ActiveTheme != nil {
		if plan.ActiveTheme.Changed {
			printSectionHeader(stdout, &printedObsidian, "Obsidian Settings")
			fmt.Fprintf(stdout, "  %s %-14s %s\n", out.change("~"), "active-theme", plan.ActiveTheme.ThemeName)
		}
	}
	if plan.Fonts != nil {
		for _, change := range plan.Fonts.Changes {
			printSectionHeader(stdout, &printedObsidian, "Obsidian Settings")
			fmt.Fprintf(stdout, "  %s %-14s %s\n", out.change("~"), change.Name, change.Value)
		}
	}
	if plan.VimMode != nil {
		if plan.VimMode.Changed {
			printSectionHeader(stdout, &printedObsidian, "Obsidian Settings")
			fmt.Fprintf(stdout, "  %s %-14s %t\n", out.change("~"), "vim-mode", plan.VimMode.Enable)
		}
	}
	if plan.ShowLineNumber != nil {
		if plan.ShowLineNumber.Changed {
			printSectionHeader(stdout, &printedObsidian, "Obsidian Settings")
			fmt.Fprintf(stdout, "  %s %-14s %t\n", out.change("~"), "line-numbers", plan.ShowLineNumber.Enable)
		}
	}

	printedFiles := false
	for _, cp := range plan.PluginSettings {
		if cp.Skipped {
			continue
		}
		if len(cp.Files) == 0 {
			continue
		}
		printSectionHeader(stdout, &printedFiles, "Files To Copy")
		fmt.Fprintf(stdout, "  %s %-24s %s\n", out.change("~"), "plugin/"+cp.PluginID, fileCount(len(cp.Files)))
		if verbose {
			fmt.Fprintf(stdout, "    source %s\n", cp.Source)
		}
	}
	if plan.Hotkeys != nil {
		if plan.Hotkeys.Changed {
			printSectionHeader(stdout, &printedFiles, "Files To Copy")
			fmt.Fprintf(stdout, "  %s %-24s %s\n", out.change("~"), "hotkeys", relativeTarget(plan, plan.Hotkeys.Target))
			if verbose {
				fmt.Fprintf(stdout, "    source %s\n", plan.Hotkeys.Source)
			}
		}
	}
	for _, cp := range plan.ObsidianSettings {
		if cp.Changed {
			printSectionHeader(stdout, &printedFiles, "Files To Copy")
			fmt.Fprintf(stdout, "  %s %-24s %s\n", out.change("~"), cp.Name, relativeTarget(plan, cp.Target))
			if verbose {
				fmt.Fprintf(stdout, "    source %s\n", cp.Source)
			}
		}
	}
	for _, cp := range plan.VaultFiles {
		if cp.Changed {
			printSectionHeader(stdout, &printedFiles, "Files To Copy")
			fmt.Fprintf(stdout, "  %s %-24s %s\n", out.change("~"), "vault-file", relativeTarget(plan, cp.Target))
			if verbose {
				fmt.Fprintf(stdout, "    source %s\n", cp.Source)
			}
		}
	}

	if verbose {
		printUnchanged(plan, stdout, out)
	}
}

func printUnchanged(plan Plan, stdout io.Writer, out textStyle) {
	printed := false
	for _, p := range plan.Themes {
		if !p.Changed() {
			printSectionHeader(stdout, &printed, "Unchanged")
			fmt.Fprintf(stdout, "  %s theme %s\n", out.same("="), p.Name)
		}
	}
	if plan.ActiveTheme != nil && !plan.ActiveTheme.Changed {
		printSectionHeader(stdout, &printed, "Unchanged")
		fmt.Fprintf(stdout, "  %s active-theme %s\n", out.same("="), plan.ActiveTheme.ThemeName)
	}
	if plan.Fonts != nil && !plan.Fonts.Changed() {
		printSectionHeader(stdout, &printed, "Unchanged")
		fmt.Fprintf(stdout, "  %s fonts\n", out.same("="))
	}
	if plan.VimMode != nil && !plan.VimMode.Changed {
		printSectionHeader(stdout, &printed, "Unchanged")
		fmt.Fprintf(stdout, "  %s vim-mode %t\n", out.same("="), plan.VimMode.Enable)
	}
	if plan.ShowLineNumber != nil && !plan.ShowLineNumber.Changed {
		printSectionHeader(stdout, &printed, "Unchanged")
		fmt.Fprintf(stdout, "  %s line-numbers %t\n", out.same("="), plan.ShowLineNumber.Enable)
	}
	if len(plan.CommunityPluginsToAdd) == 0 {
		printSectionHeader(stdout, &printed, "Unchanged")
		fmt.Fprintf(stdout, "  %s community-plugins.json\n", out.same("="))
	}
	for _, cp := range plan.PluginSettings {
		if cp.Skipped || len(cp.Files) > 0 {
			continue
		}
		printSectionHeader(stdout, &printed, "Unchanged")
		fmt.Fprintf(stdout, "  %s plugin/%s settings\n", out.same("="), cp.PluginID)
	}
	if plan.Hotkeys != nil && !plan.Hotkeys.Changed {
		printSectionHeader(stdout, &printed, "Unchanged")
		fmt.Fprintf(stdout, "  %s %s\n", out.same("="), relativeTarget(plan, plan.Hotkeys.Target))
	}
	for _, cp := range plan.ObsidianSettings {
		if cp.Changed {
			continue
		}
		printSectionHeader(stdout, &printed, "Unchanged")
		fmt.Fprintf(stdout, "  %s %s\n", out.same("="), relativeTarget(plan, cp.Target))
	}
	for _, cp := range plan.VaultFiles {
		if cp.Changed {
			continue
		}
		printSectionHeader(stdout, &printed, "Unchanged")
		fmt.Fprintf(stdout, "  %s %s\n", out.same("="), relativeTarget(plan, cp.Target))
	}
}

func printSectionHeader(stdout io.Writer, printed *bool, title string) {
	if !*printed {
		fmt.Fprintf(stdout, "\n%s\n", title)
		*printed = true
	}
}

func printWarnings(plan Plan, stdout io.Writer) {
	printWarningsWithStyle(plan, stdout, newStyle(stdout))
}

func printWarningsWithStyle(plan Plan, w io.Writer, style textStyle) {
	for _, warning := range plan.Warnings {
		fmt.Fprintf(w, "%s warning: %s\n", style.warn("!"), warning)
	}
}

func planSummary(plan Plan) string {
	changes := plan.ChangeCount()
	if changes == 0 {
		return "Plan: no changes"
	}
	if changes == 1 {
		return "Plan: 1 change"
	}
	return fmt.Sprintf("Plan: %d changes", changes)
}

func applySummary(plan Plan) string {
	changes := plan.ChangeCount()
	if changes == 0 {
		return "Applying: no changes"
	}
	if changes == 1 {
		return "Applying: 1 change"
	}
	return fmt.Sprintf("Applying: %d changes", changes)
}

func (p Plan) ChangeCount() int {
	count := len(p.PluginInstalls) + len(p.CommunityPluginsToAdd)
	for _, pluginID := range p.CommunityPluginsToAdd {
		if installPlanned(p.PluginInstalls, pluginID) {
			count--
		}
	}
	for _, themePlan := range p.Themes {
		if themePlan.Changed() {
			count++
		}
	}
	if p.ActiveTheme != nil && p.ActiveTheme.Changed {
		count++
	}
	if p.Fonts != nil {
		count += len(p.Fonts.Changes)
	}
	if p.VimMode != nil && p.VimMode.Changed {
		count++
	}
	if p.ShowLineNumber != nil && p.ShowLineNumber.Changed {
		count++
	}
	for _, cp := range p.PluginSettings {
		if !cp.Skipped && len(cp.Files) > 0 {
			count++
		}
	}
	if p.Hotkeys != nil && p.Hotkeys.Changed {
		count++
	}
	for _, cp := range p.ObsidianSettings {
		if cp.Changed {
			count++
		}
	}
	for _, cp := range p.VaultFiles {
		if cp.Changed {
			count++
		}
	}
	return count
}

func installPlanned(plans []install.Plan, pluginID string) bool {
	for _, p := range plans {
		if p.PluginID == pluginID {
			return true
		}
	}
	return false
}

func unmanagedEnabledPlugins(v vault.Vault, configured []string) ([]string, error) {
	enabled, err := v.ReadEnabledPlugins()
	if err != nil {
		return nil, err
	}
	configuredSet := map[string]bool{}
	for _, pluginID := range configured {
		configuredSet[pluginID] = true
	}
	var unmanaged []string
	for _, pluginID := range enabled {
		if !configuredSet[pluginID] {
			unmanaged = append(unmanaged, pluginID)
		}
	}
	sort.Strings(unmanaged)
	return unmanaged, nil
}

func fileCount(count int) string {
	if count == 1 {
		return "1 file"
	}
	return fmt.Sprintf("%d files", count)
}

func relativeTarget(plan Plan, target string) string {
	rel, err := filepath.Rel(plan.VaultPath, target)
	if err != nil || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
		return target
	}
	return rel
}

type textStyle struct {
	enabled bool
}

func newStyle(w io.Writer) textStyle {
	if os.Getenv("NO_COLOR") != "" {
		return textStyle{}
	}
	file, ok := w.(*os.File)
	if !ok {
		return textStyle{}
	}
	info, err := file.Stat()
	if err != nil || info.Mode()&os.ModeCharDevice == 0 {
		return textStyle{}
	}
	return textStyle{enabled: true}
}

func (s textStyle) heading(value string) string { return s.wrap("1", value) }
func (s textStyle) add(value string) string     { return s.wrap("32", value) }
func (s textStyle) change(value string) string  { return s.wrap("33", value) }
func (s textStyle) same(value string) string    { return s.wrap("2", value) }
func (s textStyle) warn(value string) string    { return s.wrap("31", value) }

func (s textStyle) wrap(code string, value string) string {
	if !s.enabled {
		return value
	}
	return "\x1b[" + code + "m" + value + "\x1b[0m"
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
