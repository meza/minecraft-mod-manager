# cmd/mmm/list

This package implements `mmm list`: render the configured mods and whether each one appears to be installed locally.

## Start with the behavior docs

- User guide: `docs/commands/list.md`
- Command spec: `docs/specs/list.md`

## Code map

- `cmd/mmm/list/list.go`: command implementation (read config, read lock, compute installed status, render view)
- `cmd/mmm/list/config_prompt.go`: Bubble Tea prompt for missing-config recovery
- `cmd/mmm/list/output_model.go`: Bubble Tea output-only model for list rendering
- `cmd/mmm/list/view_snapshot_test.go`: snapshot tests for the rendered list view
- `cmd/mmm/list/list_test.go`: behavior tests for installed detection and lock handling

## Installed detection

An entry is considered "installed" when:

- there is a matching lock entry (same platform + ID), and
- the lock entry has a `fileName`, and
- the lock entry has a `hash`, and
- that file exists in the configured mods folder with a matching hash

If the file exists but the hash does not match, the entry is reported as not installed with a distinct hash mismatch message.

## Interactive vs unattended behavior

`list` renders output through Bubble Tea in all execution contexts, using an output-only program when no prompts are required.

When config is missing and prompts are allowed, `list` shows a Bubble Tea confirm prompt, runs the `init` interactive flow on acceptance, and resumes listing.

## Testing and snapshots

See `CONTRIBUTING.md` for required test/coverage checks and snapshot update instructions.

Snapshots live under `cmd/mmm/list/__snapshots__/` and tests set `MMM_TEST=true` so i18n output is stable.
