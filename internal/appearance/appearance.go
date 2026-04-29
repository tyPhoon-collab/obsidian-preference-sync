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
