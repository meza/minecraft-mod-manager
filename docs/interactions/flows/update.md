# Flow: update

## User goal and success conditions

You want to update your configured mods to newer compatible versions without re-adding everything.

Success looks like:
- Updated mods have new jars on disk
- `modlist-lock.json` is updated deterministically
- Pinned mods are not updated

## Entry points

- Command: `mmm update`
- User guide: `docs/commands/update.md`
- Behavior spec: `docs/specs/update.md`

## Primary flow

1. You run `mmm update`.
2. MMM runs `install` first to ensure consistency.
3. MMM checks each unpinned mod for newer compatible releases.
4. MMM swaps jars and updates lock and config names when needed.

## Interactive behavior

This command does not prompt.
When TUI is allowed it can use a Bubble Tea log wrapper for stable interactive output.

## References in code

- Command implementation: `cmd/mmm/update/update.go`
- Install prerequisite: `cmd/mmm/install/install.go`

