# Flow: add

## User goal and success conditions

You want to add a mod to your config and download a compatible jar into your mods folder.

Success looks like:
- A new entry exists in `modlist.json`
- A matching entry exists in `modlist-lock.json`
- The jar exists on disk in the configured mods folder

## Entry points

- Command: `mmm add <platform> <id>`
- User guide: `docs/commands/add.md`
- Behavior spec: `docs/specs/add.md`

## Primary flow

1. You run `mmm add <platform> <id>`.
2. MMM reads config and lock, creating them when allowed.
3. MMM resolves a compatible file and downloads it.
4. MMM writes config and lock updates.

## Key alternate paths

Interactive recovery exists for expected errors when prompts and TUI are allowed:
- Unknown platform
- Mod not found
- No compatible file found

## Non-interactive behavior

When `--non-interactive` is set or the session is not a TTY:
- Do not launch interactive recovery
- Fail with an actionable error

## Feedback and recovery

- In TUI recovery, `esc` goes back and `ctrl+c` aborts.
- Messages should make it clear which platform and id were attempted.

## References in code

- Command implementation: `cmd/mmm/add/add.go`
- Recovery TUI: `cmd/mmm/add/tui.go`
- TTY gating: `internal/tui/terminal.go`

