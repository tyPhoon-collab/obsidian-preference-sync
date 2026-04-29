package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const CommunityPluginsURL = "https://raw.githubusercontent.com/obsidianmd/obsidian-releases/master/community-plugins.json"
const CommunityThemesURL = "https://raw.githubusercontent.com/obsidianmd/obsidian-releases/master/community-css-themes.json"

type Plugin struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Author      string `json:"author"`
	Description string `json:"description"`
	Repo        string `json:"repo"`
}

type Registry struct {
	plugins map[string]Plugin
}

type Theme struct {
	Name   string `json:"name"`
	Author string `json:"author"`
	Repo   string `json:"repo"`
}

type ThemeRegistry struct {
	themes map[string]Theme
}

func Fetch(ctx context.Context) (Registry, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, CommunityPluginsURL, nil)
	if err != nil {
		return Registry{}, err
	}
	req.Header.Set("User-Agent", "obsidian-preference-sync")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return Registry{}, fmt.Errorf("fetch Obsidian community registry: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Registry{}, fmt.Errorf("fetch Obsidian community registry: HTTP %s", resp.Status)
	}

	var plugins []Plugin
	if err := json.NewDecoder(resp.Body).Decode(&plugins); err != nil {
		return Registry{}, fmt.Errorf("parse Obsidian community registry: %w", err)
	}
	byID := make(map[string]Plugin, len(plugins))
	for _, plugin := range plugins {
		byID[plugin.ID] = plugin
	}
	return Registry{plugins: byID}, nil
}

func (r Registry) Lookup(id string) (Plugin, bool) {
	plugin, ok := r.plugins[id]
	return plugin, ok
}

func FetchThemes(ctx context.Context) (ThemeRegistry, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, CommunityThemesURL, nil)
	if err != nil {
		return ThemeRegistry{}, err
	}
	req.Header.Set("User-Agent", "obsidian-preference-sync")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return ThemeRegistry{}, fmt.Errorf("fetch Obsidian community theme registry: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ThemeRegistry{}, fmt.Errorf("fetch Obsidian community theme registry: HTTP %s", resp.Status)
	}

	var themes []Theme
	if err := json.NewDecoder(resp.Body).Decode(&themes); err != nil {
		return ThemeRegistry{}, fmt.Errorf("parse Obsidian community theme registry: %w", err)
	}
	byName := make(map[string]Theme, len(themes))
	for _, theme := range themes {
		byName[theme.Name] = theme
	}
	return ThemeRegistry{themes: byName}, nil
}

func (r ThemeRegistry) Lookup(name string) (Theme, bool) {
	theme, ok := r.themes[name]
	return theme, ok
}
