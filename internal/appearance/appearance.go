package appearance

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"obsidian-preference-sync/internal/fileutil"
	"obsidian-preference-sync/internal/vault"
)

type Plan struct {
	SourcePath string
	ThemeName  string
	Changed    bool
}

type FontChange struct {
	Name  string
	Key   string
	Value string
}

type Fonts struct {
	Interface string
	Text      string
	Monospace string
}

type FontPlan struct {
	SourcePath string
	Fonts      Fonts
	Changes    []FontChange
}

func BuildPlan(v vault.Vault, themeName string) (Plan, error) {
	path := filepath.Join(v.ObsidianDir, "appearance.json")
	current, err := readAppearance(path)
	if err != nil {
		return Plan{}, err
	}
	return Plan{
		SourcePath: path,
		ThemeName:  themeName,
		Changed:    current["cssTheme"] != themeName,
	}, nil
}

func BuildFontPlan(v vault.Vault, fonts Fonts) (FontPlan, error) {
	path := filepath.Join(v.ObsidianDir, "appearance.json")
	current, err := readAppearance(path)
	if err != nil {
		return FontPlan{}, err
	}
	plan := FontPlan{
		SourcePath: path,
		Fonts:      fonts,
	}
	for _, change := range fontChanges(fonts) {
		if current[change.Key] != change.Value {
			plan.Changes = append(plan.Changes, change)
		}
	}
	return plan, nil
}

func Apply(plan Plan) error {
	if !plan.Changed {
		return nil
	}
	current, err := readAppearance(plan.SourcePath)
	if err != nil {
		return err
	}
	current["cssTheme"] = plan.ThemeName
	return fileutil.WriteJSONAtomic(plan.SourcePath, current, 0o644)
}

func ApplyFonts(plan FontPlan) error {
	if !plan.Changed() {
		return nil
	}
	current, err := readAppearance(plan.SourcePath)
	if err != nil {
		return err
	}
	for _, change := range fontChanges(plan.Fonts) {
		current[change.Key] = change.Value
	}
	return fileutil.WriteJSONAtomic(plan.SourcePath, current, 0o644)
}

func (p FontPlan) Changed() bool {
	return len(p.Changes) > 0
}

func fontChanges(fonts Fonts) []FontChange {
	changes := []FontChange{}
	if fonts.Interface != "" {
		changes = append(changes, FontChange{Name: "interface-font", Key: "interfaceFontFamily", Value: fonts.Interface})
	}
	if fonts.Text != "" {
		changes = append(changes, FontChange{Name: "text-font", Key: "textFontFamily", Value: fonts.Text})
	}
	if fonts.Monospace != "" {
		changes = append(changes, FontChange{Name: "monospace-font", Key: "monospaceFontFamily", Value: fonts.Monospace})
	}
	return changes
}

func readAppearance(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[string]any{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read appearance.json: %w", err)
	}
	if len(data) == 0 {
		return map[string]any{}, nil
	}
	var appearance map[string]any
	if err := json.Unmarshal(data, &appearance); err != nil {
		return nil, fmt.Errorf("parse appearance.json: %w", err)
	}
	if appearance == nil {
		appearance = map[string]any{}
	}
	return appearance, nil
}
