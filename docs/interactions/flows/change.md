# Flow: change

## User goal and success conditions

You want to move your configuration to a different Minecraft version and reinstall your mod set for that version.

Success looks like:
- The configured `gameVersion` changes to the target version
- Currently installed mod jars are removed
- A new compatible set of jars is installed

## Entry points

- Command: `mmm change [game_version]`
- User guide: `docs/commands/change.md`
- Behavior spec: `docs/specs/change.md`

## Primary flow

1. You run `mmm change [game_version]`.
2. MMM resolves the target version, defaulting to `latest` when not provided.
3. MMM runs the same compatibility checks as `mmm test` unless `--force` is set.
4. MMM removes installed jars for the current lock and clears `modlist-lock.json`.
5. MMM writes the new `gameVersion` to `modlist.json`.
6. MMM runs `mmm install` to download compatible jars for the new version.

## Key alternate paths

- If you attempt to change to the current version, the command exits with code `2` and makes no changes.
- If some mods do not support the target version, the command fails with code `1` unless `--force` is set.
- With `--force`, MMM proceeds even if some mods fail to install, and unsupported mods are skipped during install.

## Non-interactive behavior

This command does not prompt.
When `--non-interactive` is set, MMM must not launch any TUI wrappers.

## Feedback and recovery

- Errors should explain which step failed, for example version check versus install.
- When TUI is allowed, the command can use a Bubble Tea log wrapper for stable interactive output.

## References in code

- Command implementation: `cmd/mmm/change/change.go`
- Runner: `cmd/mmm/change/run.go`
- TTY gating: `internal/tui/terminal.go`

