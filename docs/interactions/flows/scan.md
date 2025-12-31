# Flow: scan

## User goal and success conditions

You want to identify jar files in your mods folder that are not managed by MMM, and optionally add them to your config and lock.

Success looks like:
- You see which files are unmanaged
- You can add safe results to config and lock without guessing

## Entry points

- Command: `mmm scan`
- User guide: `docs/commands/scan.md`
- Behavior spec: `docs/specs/scan.md`

## Primary flow

1. You run `mmm scan`.
2. MMM lists candidate jar files and filters out ignored and managed files, applying `.mmmignore` rules.
3. MMM tries to identify each candidate by hash.
4. MMM prints matches, unknown files, and unsure files.

## Key alternate paths

- If config is missing and prompts are allowed, MMM can ask to run interactive init and then continue.
- If `--add` is set, MMM persists results without prompting, unless there are unsure files.

## Non-interactive behavior

When `--non-interactive` is set:
- Do not prompt to init
- Do not prompt to persist
- Only write when `--add` is set and there are no unsure files

## References in code

- Command implementation: `cmd/mmm/scan/scan.go`
- Prompt gating: `internal/tui/terminal.go`
