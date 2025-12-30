# Flow: remove

## User goal and success conditions

You want to remove one or more mods from your configuration, and delete their jar files when present.

Success looks like:
- The targeted mods are removed from `modlist.json`
- The targeted mods are removed from `modlist-lock.json`
- The jar files are deleted when they exist

## Entry points

- Command: `mmm remove <mods...>`
- User guide: `docs/commands/remove.md`
- Behavior spec: `docs/specs/remove.md`

## Primary flow

1. You run `mmm remove <mods...>`.
2. MMM resolves each lookup against config.
3. MMM deletes jar files when possible and updates config and lock.

## Key alternate paths

- `--dry-run` prints what would be removed and makes no changes.
- Missing jars should not block config and lock updates.

## Non-interactive behavior

This command is designed to be non-interactive.

## References in code

- Command implementation: `cmd/mmm/remove/remove.go`

