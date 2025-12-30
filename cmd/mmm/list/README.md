# cmd/mmm/list

This package implements `mmm list`: render the configured mods and whether each one appears to be installed locally.

## Start with the behavior docs

- User guide: `docs/commands/list.md`
- Command spec: `docs/specs/list.md`

## Code map

- `cmd/mmm/list/list.go`: command implementation (read config, read lock, compute installed status, render view)
- `cmd/mmm/list/tui.go`: minimal Bubble Tea wrapper used to keep output consistent in interactive mode
- `cmd/mmm/list/tui_snapshot_test.go`: snapshot tests for the rendered list view
- `cmd/mmm/list/list_test.go`: behavior tests for installed detection and lock handling

## Installed detection

An entry is considered "installed" when:

- there is a matching lock entry (same platform + ID), and
- the lock entry has a `fileName`, and
- the lock entry has a `hash`, and
- that file exists in the configured mods folder with a matching hash

If the file exists but the hash does not match, the entry is reported as not installed with a distinct hash mismatch message.

## Interactive vs non-interactive behavior

`list` will render through Bubble Tea only when stdin and stdout are terminals and `--non-interactive` is not set (see `internal/tui.ShouldUseTUI`).

When it runs in TUI mode and the list is empty, it also logs the rendered view to stdout. This keeps the result visible even though Bubble Tea exits immediately.

## Testing and snapshots

See `CONTRIBUTING.md` for required test/coverage checks and snapshot update instructions.

Snapshots live under `cmd/mmm/list/__snapshots__/` and tests set `MMM_TEST=true` so i18n output is stable.
