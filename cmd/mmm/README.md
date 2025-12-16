# cmd/mmm

This directory contains the cobra command tree for the `mmm` CLI.

If you are adding a new command, start here: the root command wires global flags, help behavior, and subcommands.

## Start with the behavior docs

For command behavior, prefer these docs over reverse-engineering the code:

- Command reference: `docs/commands/README.md`
- Command specs: `docs/specs/README.md`

They describe what each command should do. This package is where that behavior gets wired into cobra.

## Code map

- `cmd/mmm/root.go`: root command, global flags, help templates, and subcommand registration
- `cmd/mmm/root_test.go`: basic smoke test coverage for root command behavior
- `cmd/mmm/<command>`: implementation of each subcommand (each should have its own README)

## Global flags and help behavior

`root.go` owns a few cross-cutting behaviors that affect every command:

- Persistent flags: `--config`, `--quiet`, `--debug`
- Help localization: overrides the default help flag text and help command text using `internal/i18n`
- Help footer: appends a "more info" footer using `internal/environment.HelpURL()`
- Usage formatting: wraps flag usage to terminal width so help output stays readable

## Adding a new command package

To add a new command:

1. Create a new package under `cmd/mmm/<command>`.
2. Provide `Command() *cobra.Command`.
3. Register it in `cmd/mmm/root.go`.
4. Add:
   - a user-facing page under `docs/commands/<command>.md`
   - a behavior spec under `docs/specs/<command>.md`
   - a maintainer README next to the code (`cmd/mmm/<command>/README.md`)

## Tests

Run from the repo root:

```bash
make test
```

