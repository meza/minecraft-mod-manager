# Flow: test

## User goal and success conditions

You want to know if your configured mods support a target Minecraft version before you change anything.

Success looks like:
- You see which mods block an upgrade
- Exit codes are usable in scripts

## Entry points

- Command: `mmm test [game_version]`
- User guide: `docs/commands/test.md`
- Behavior spec: `docs/specs/test.md`

## Primary flow

1. You run `mmm test [game_version]`.
2. MMM resolves a target version, defaulting to latest when possible.
3. MMM checks each configured mod for a compatible release.
4. MMM prints results and returns an exit code.

## Non-interactive behavior

In non-interactive environments, MMM must fail fast if it cannot determine the latest version and you did not provide one.

## References in code

- Command implementation: `cmd/mmm/test/test.go`
- TTY gating: `internal/tui/terminal.go`

