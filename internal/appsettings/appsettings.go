package appsettings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"obsidian-preference-sync/internal/fileutil"
	"obsidian-preference-sync/internal/vault"
)

type VimModePlan struct {
	Path    string
	Enable  bool
	Changed bool
}

type ShowLineNumberPlan struct {
	Path    string
	Enable  bool
	Changed bool
}

func BuildVimModePlan(v vault.Vault, enable bool) (VimModePlan, error) {
	path := filepath.Join(v.ObsidianDir, "app.json")
	current, err := readAppSettings(path)
	if err != nil {
		return VimModePlan{}, err
	}
	return VimModePlan{
		Path:    path,
		Enable:  enable,
		Changed: current["vimMode"] != enable,
	}, nil
}

func BuildShowLineNumberPlan(v vault.Vault, enable bool) (ShowLineNumberPlan, error) {
	path := filepath.Join(v.ObsidianDir, "app.json")
	current, err := readAppSettings(path)
	if err != nil {
		return ShowLineNumberPlan{}, err
	}
	return ShowLineNumberPlan{
		Path:    path,
		Enable:  enable,
		Changed: current["showLineNumber"] != enable,
	}, nil
}

func ApplyVimMode(plan VimModePlan) error {
	if !plan.Changed {
		return nil
	}
	current, err := readAppSettings(plan.Path)
	if err != nil {
		return err
	}
	current["vimMode"] = plan.Enable
	return fileutil.WriteJSONAtomic(plan.Path, current, 0o644)
}

func ApplyShowLineNumber(plan ShowLineNumberPlan) error {
	if !plan.Changed {
		return nil
	}
	current, err := readAppSettings(plan.Path)
	if err != nil {
		return err
	}
	current["showLineNumber"] = plan.Enable
	return fileutil.WriteJSONAtomic(plan.Path, current, 0o644)
}

func readAppSettings(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[string]any{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read app.json: %w", err)
	}
	if len(data) == 0 {
		return map[string]any{}, nil
	}
	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("parse app.json: %w", err)
	}
	if settings == nil {
		settings = map[string]any{}
	}
	return settings, nil
}
