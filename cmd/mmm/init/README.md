# cmd/mmm/init

This package implements `mmm init`: create a new `modlist.json` and `modlist-lock.json` by collecting the required settings (loader, game version, release types, and mods folder).

## Start with the behavior docs

- Product intent: [`docs/intent.md`](../../../docs/intent.md)
- User guide: [`docs/commands/init.md`](../../../docs/commands/init.md)
- Terminal interaction implementation guide: [`docs/guide-to-working-with-the-terminal.md`](../../../docs/guide-to-working-with-the-terminal.md)

## Code map

- `cmd/mmm/init/init.go`: cobra wiring, flag parsing, and `initWithDeps` (writes the modlist and lockfile)
- `cmd/mmm/init/interactive_flow.go`: interactive Bubble Tea flow that asks for values and handles overwrite/confirm steps
- `cmd/mmm/init/*Model*.go`: individual prompt models (loader, game version, release types, mods folder)
- `cmd/mmm/init/interactive_flow_snapshot_test.go`: snapshot tests for the interactive flow
- `cmd/mmm/init/init_test.go`: behavior tests (flags, overwrite flow, validation)

## Execution flow

`init` is designed to be both scriptable and friendly:

- If you provide all required values (including the default mods folder), the command writes the files without prompting.
- If required values are missing and stdin/stdout are terminals with prompts allowed, it launches an interactive flow to collect values and confirm writing.
- If the modlist path already exists, the flow asks whether to overwrite or choose a new path.
- In `--unattended` or non-TTY mode, it never prompts and fails fast when required inputs are missing or the modlist already exists (unless `--force` is set).

After inputs are finalized (via flags or the interactive flow), `initWithDeps` writes `modlist.json` and an empty `modlist-lock.json`. Success output is emitted afterward through a Bubble Tea output-only program.

## Testing and snapshots

See `CONTRIBUTING.md` for required test/coverage checks and snapshot update instructions.

Snapshots live under `cmd/mmm/init/__snapshots__/` and tests set `MMM_TEST=true` so i18n output is stable.
