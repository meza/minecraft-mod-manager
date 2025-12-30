# Flow: list

## User goal and success conditions

You want to see which mods are configured and whether they are installed correctly.

Success looks like:
- You can tell which mods are installed
- You can tell what to do when a mod is missing or mismatched

## Entry points

- Command: `mmm list`
- User guide: `docs/commands/list.md`
- Behavior spec: `docs/specs/list.md`

## Primary flow

1. You run `mmm list`.
2. MMM reads config and lock.
3. MMM checks local files and hashes.
4. MMM renders a view that shows installed and missing states.

## Interactive behavior

When TUI is allowed, `list` can render through Bubble Tea.

## Non-interactive behavior

When `--non-interactive` is set:
- Do not prompt
- Still provide a readable, non-colorized output

## References in code

- Command implementation: `cmd/mmm/list/list.go`
- TUI wrapper: `cmd/mmm/list/tui.go`
- TTY gating: `internal/tui/terminal.go`

