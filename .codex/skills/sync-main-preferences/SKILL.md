---
name: sync-main-preferences
description: Synchronize this repository's examples preference pack with a user-specified reference Obsidian vault. Use when Codex is asked to compare, align, or update CLI-supported example preferences against a reference vault, including plugins, plugin settings, Obsidian settings, and vault files, then verify with obsidian-preference-sync --dry-run.
---

# Sync Main Preferences

## Overview

Use the user-specified reference vault for `examples/`, while keeping the repository as the only edit target. If the current project instructions name a default reference vault, use that value without copying the private path into committed skill files. First determine which CLI-supported preference category the user wants to sync; if the request is ambiguous, ask them to choose before editing.

## Workflow

1. Inspect the repo state first:

   ```sh
   git status --short
   rg --files
   sed -n '1,220p' examples/config.toml
   ```

2. Resolve the reference vault path from the user's request or project instructions. In commands below, substitute it for `$REFERENCE_VAULT`.

3. Classify the requested sync scope. Supported categories are:

   - Plugins: `plugins` entries in `examples/config.toml`, compared with `$REFERENCE_VAULT/.obsidian/community-plugins.json`
   - Plugin settings: `[plugin_settings]` entries and files under `examples/plugin-settings/`
   - Hotkeys: `hotkeys` file, usually `examples/obsidian-settings/hotkeys.json`
   - Command palette: `command_palette` file, usually `examples/obsidian-settings/command-palette.json`
   - Vault files: `[[vault_files]]` entries, usually files under `examples/vault-files/`
   - Supported app or appearance fields: `vim_mode`, `show_line_number`, `[fonts]`, `themes`, and `active_theme`

   If the user asks broadly to "sync everything", "align examples", "reflect Obsidian changes", or similar without naming a category, ask which supported category they want to sync before editing. Do not infer that every reference-vault difference should be imported.

4. Identify comparison targets from `examples/config.toml`:

   - `hotkeys` maps to `$REFERENCE_VAULT/.obsidian/hotkeys.json`
   - `command_palette` maps to `$REFERENCE_VAULT/.obsidian/command-palette.json`
   - `[plugin_settings] <plugin-id> = "plugin-settings/<plugin-id>"` maps to `$REFERENCE_VAULT/.obsidian/plugins/<plugin-id>/`
   - `[[vault_files]] target = "path"` maps to `$REFERENCE_VAULT/path`
   - `plugins` maps to enabled plugin IDs in `$REFERENCE_VAULT/.obsidian/community-plugins.json`
   - `themes` and `active_theme` map to `$REFERENCE_VAULT/.obsidian/appearance.json` plus installed theme directories
   - `vim_mode` and `show_line_number` map to `$REFERENCE_VAULT/.obsidian/app.json`
   - `[fonts]` maps to font fields in `$REFERENCE_VAULT/.obsidian/appearance.json`

5. Compare before editing using the simplest appropriate method for the requested category. Prefer a targeted comparison over broad vault scans. For JSON, account for formatting or key-order noise when it matters; for plugin membership, compare IDs as a set.

6. Report the discovered diff before editing when the user asks to "first check" or similar. Include the specific plugin ID, command ID, setting key, or file path, not just "files differ".

7. Edit only repository files under `examples/`. Do not edit the reference vault; it is the source of truth for this workflow.

8. Keep changes narrow:

   - For JSON, preserve existing formatting unless copying a whole file is explicitly intended.
   - Add only the missing keys or setting values needed to match the reference vault.
   - If the goal is exact match, make `diff -u` clean, including trailing newline differences.
   - When plugin settings changed but the plugin is not listed in `plugins`, ask whether to sync the plugin, the settings, or both.
   - When a reference-vault `.obsidian/*.json` change is not represented by current CLI config fields, report it as unsupported by the current CLI instead of adding ad hoc files.

## Validation

After editing:

1. Validate changed JSON files:

   ```sh
   jq empty examples/obsidian-settings/hotkeys.json
   ```

2. Re-run exact diffs against the reference vault for all changed files:

   ```sh
   diff -u examples/obsidian-settings/hotkeys.json "$REFERENCE_VAULT/.obsidian/hotkeys.json"
   ```

3. Run the project dry run against the reference vault:

   ```sh
   go run ./cmd/obsidian-preference-sync --vault "$REFERENCE_VAULT" --config ./examples/config.toml --dry-run
   ```

   This command fetches the Obsidian community plugin registry from GitHub. If it fails because network access is sandboxed, rerun the same command with approval for network access.

4. Treat `Plan: no changes` as the success condition. Warnings about enabled plugins not listed in `examples/config.toml` are acceptable unless the user asks to reconcile plugin membership.

5. Finish by reporting:

   - Files changed
   - Whether exact diffs are clean
   - The `--dry-run` result
   - Any warnings separately from planned changes
