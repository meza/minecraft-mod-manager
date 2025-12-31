# Flow: install

## User goal and success conditions

You want to ensure your mods folder matches your config and lock file.

Success looks like:
- Each configured mod has a matching jar on disk
- Lock entries match the file hash

## Entry points

- Command: `mmm install`
- User guide: `docs/commands/install.md`
- Behavior spec: `docs/specs/install.md`

## Primary flow

1. You run `mmm install`.
2. MMM reads config and lock.
3. MMM scans the mods folder for unmanaged jar files, applying `.mmmignore` rules.
4. If unmanaged files exist and MMM cannot resolve them, MMM fails and suggests running `mmm scan`.
5. MMM downloads any missing or mismatched jars based on lock or resolved latest compatible files.

## Interactive behavior

When TUI is allowed, `install` can run with a Bubble Tea log wrapper for stable interactive output.

## Non-interactive behavior

When `--non-interactive` is set:
- Do not prompt
- Continue to behave deterministically for scripts

## References in code

- Command implementation: `cmd/mmm/install/install.go`
- TTY gating: `internal/tui/terminal.go`
