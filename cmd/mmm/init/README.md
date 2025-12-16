# cmd/mmm/init

This package implements `mmm init`: create a new `modlist.json` and `modlist-lock.json` by collecting the required settings (loader, game version, release types, and mods folder).

## Start with the behavior docs

- User guide: `docs/commands/init.md`
- Command spec: `docs/specs/init.md`

## Code map

- `cmd/mmm/init/init.go`: cobra wiring, flag parsing, and `initWithDeps` (writes config and lock)
- `cmd/mmm/init/tui.go`: interactive Bubble Tea flow that asks for any missing values
- `cmd/mmm/init/*Model*.go`: individual prompt models (loader, game version, release types, mods folder)
- `cmd/mmm/init/tui_snapshot_test.go`: snapshot tests for the interactive flow
- `cmd/mmm/init/init_test.go`: behavior tests (flags, overwrite flow, validation)

## Execution flow

`init` is designed to be both scriptable and friendly:

- If you provide all required flags, the command is non-interactive and writes the files.
- If you omit required flags and stdout/stderr are terminals, it launches a TUI to collect the missing values.

After inputs are finalized, `initWithDeps`:

1. Validates the mods folder exists and is a directory (relative to the config file directory unless absolute).
2. Resolves `latest` to the current Minecraft release version when needed.
3. Validates the Minecraft version against the Mojang manifest (see `internal/minecraft`), but allows offline use where appropriate.
4. Writes `modlist.json` and an empty `modlist-lock.json`.

## Overwrite behavior (non-TUI prompt)

Even when the TUI is not used, `initWithDeps` may prompt on stdout/stderr if the config file already exists and `--quiet` is not set:

- confirm overwrite, or
- enter a new config path

This prompt is handled by `terminalPrompter` in `init.go` and is separate from the Bubble Tea flow.

## Testing and snapshots

Run the full suite from the repo root:

```bash
make test
```

If you change TUI rendering, update snapshots with:

```bash
UPDATE_SNAPS=true make test
```

Snapshots live under `cmd/mmm/init/__snapshots__/` and tests set `MMM_TEST=true` so i18n output is stable.

