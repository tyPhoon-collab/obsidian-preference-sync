package config

var DangerousPluginSettings = map[string]bool{
	"obsidian-git":      true,
	"obsidian-livesync": true,
	"selfhost-livesync": true,
	"copilot":           true,
	"vim-im-select":     true,
}

func DangerousSettingsIDs(settings map[string]string) []string {
	var ids []string
	for id := range settings {
		if DangerousPluginSettings[id] {
			ids = append(ids, id)
		}
	}
	return ids
}
