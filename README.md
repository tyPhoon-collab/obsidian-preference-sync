# obsidian-preference-sync

`obsidian-preference-sync` is a small CLI for synchronizing selected Obsidian community plugins and explicitly selected plugin settings across multiple vaults or Macs.

It does not synchronize vault content. It only touches the plugin IDs and plugin settings that are explicitly listed in the config file.

## Safety policy

- `community-plugins.json` is updated by upsert only.
- Existing enabled plugin IDs are never removed.
- Only configured plugin IDs are installed.
- Only configured plugin settings directories are copied.
- `core-plugins.json`, `workspace.json`, and `workspace-mobile.json` are not touched.
- Dangerous plugin settings are rejected unless `--allow-dangerous` is passed.

Dangerous plugin settings:

- `obsidian-git`
- `obsidian-livesync`
- `selfhost-livesync`
- `copilot`
- `vim-im-select`

## Usage

```sh
obsidian-preference-sync \
  --vault "$HOME/Documents/Obsidian Vault" \
  --config ./examples/config.toml \
  --dry-run

obsidian-preference-sync \
  --vault "$HOME/Documents/Obsidian Vault" \
  --config ./examples/config.toml
```

`--dry-run` prints planned changes without writing and exits with code `0`.

`--check` is also available for CI-style drift detection. It prints planned changes and exits with code `1` when changes would be made.

Normal execution prints each change as it is applied. Pass `--verbose` to include extra detail such as individual copied settings files.
When changes are applied, the CLI reminds you to restart Obsidian so plugin and setting changes are fully loaded.

## Config

```toml
plugins = [
  "obsidian-linter",
  "easy-typing-obsidian",
]

themes = [
  "Primary",
]

active_theme = "Primary"

hotkeys = "obsidian-settings/hotkeys.json"

[plugin_settings]
obsidian-linter = "plugin-settings/obsidian-linter"
```

Relative `plugin_settings` paths are resolved from the config file directory. `~/...` is expanded to the current user's home directory.
`hotkeys` uses the same path resolution rules and copies to `.obsidian/hotkeys.json`.

`themes` installs or updates only the named community themes. Theme files are copied to `.obsidian/themes/<theme-name>/`.
`active_theme` is optional. When set, it must also be listed in `themes`, and only the `cssTheme` field in `.obsidian/appearance.json` is updated.

When copying plugin settings, these names are excluded:

- `main.js`
- `manifest.json`
- `styles.css`
- `node_modules`

Plugin settings are copied file-by-file from the configured source directory into the plugin directory in the target vault.
If a configured settings file already exists in the vault with different content, it is overwritten by the source file.
Files that exist only in the vault are not deleted.

## Development

This repository uses `mise` to define the Go toolchain.

```sh
mise trust
mise install
```

```sh
just fmt
just test
just build
```

`just check` runs format, tests, and build.

## Release

Releases are created by GitHub Actions when a version tag is pushed.
The tag is the source of truth for the release version.

Before tagging, run a local snapshot build:

```sh
just release-check
just release-snapshot
```

To publish a release:

```sh
just release v0.1.0
```

The release task fetches remote tags, shows the latest version tag, rejects non-`vX.Y.Z` versions, and stops if the requested version is not greater than the latest tag.
The release workflow uses GoReleaser to publish a GitHub Release with archives for macOS, Linux, and Windows on amd64 and arm64, plus `checksums.txt`.
Release artifacts are generated under `dist/` locally and are not committed.
