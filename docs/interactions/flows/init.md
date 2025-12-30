# Flow: init

## User goal and success conditions

You want to create a valid MMM configuration so you can manage mods in a folder.

Success looks like:
- `modlist.json` exists at the configured path
- `modlist-lock.json` exists at the same location
- The configuration is valid and points at an existing mods folder

## Entry points

- Command: `mmm init`
- User guide: `docs/commands/init.md`
- Behavior spec: `docs/specs/init.md`

## Primary flow

1. You run `mmm init` in a terminal.
2. MMM collects required values:
   - loader
   - game version
   - release types
   - mods folder
3. MMM validates the inputs and writes config and lock.

## Key alternate paths

- If you provide all required flags, MMM runs without prompting.
- If the config path already exists and TUI is not used, MMM may prompt to overwrite or choose another path.

## Non-interactive behavior

When `--non-interactive` is set:
- Do not prompt
- Fail fast if required input is missing

## Feedback and recovery

- Errors should explain what is invalid and how to fix it.
- Cancellation in the TUI should not write partial files.

## Interaction constraints

- Keyboard-only flow in the TUI
- Clear defaults for prompts

## References in code

- Command wiring: `cmd/mmm/init/init.go`
- TUI flow: `cmd/mmm/init/tui.go`
- TTY gating: `internal/tui/terminal.go`

