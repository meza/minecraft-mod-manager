# cmd/mmm/list

This package implements `mmm list`: render the mod configs and report whether a matching local file is present for each mod config's locked artifact.

## Start with the behavior docs

- Product intent: [`docs/intent.md`](../../../docs/intent.md)
- User guide: [`docs/commands/list.md`](../../../docs/commands/list.md)

## Code map

- `cmd/mmm/list/list.go`: command implementation (read the modlist and lockfile, compute installed status, render the view)
- `cmd/mmm/list/config_prompt.go`: Bubble Tea prompt for missing-modlist recovery
- `cmd/mmm/list/output_model.go`: Bubble Tea output-only model for list rendering
- `cmd/mmm/list/view_snapshot_test.go`: snapshot tests for the rendered list view
- `cmd/mmm/list/list_test.go`: behavior tests for installed detection and lock handling

## Installed detection

A mod config's locked artifact is considered "installed" when:

- there is a matching lock entry (same platform + ID), and
- the lock entry has a `fileName`, and
- the lock entry has a `hash`, and
- the named local file exists in the mods directory with a matching hash

If the local file exists but its hash does not match the locked artifact, the locked artifact is reported as not installed with a distinct hash mismatch message.

## Current presentation behavior

`list` renders output through Bubble Tea in all execution contexts, using an output-only program when no prompts are required.

When the modlist is missing and prompts are allowed, `list` shows a Bubble Tea confirm prompt, runs the `init` interactive flow on acceptance, and resumes listing.

This describes the current implementation. The target [execution profiles](../../../docs/intent.md#execution-modes-and-operator-intent) require explicit unattended execution to remain plain and append-only even on a TTY, and require plain line-oriented questions where prompting is available without control sequences. That routing is not complete yet. Supplying complete arguments can remove the need for a prompt but does not select unattended execution.

## Testing and snapshots

See `CONTRIBUTING.md` for required test/coverage checks and snapshot update instructions.

Snapshots live under `cmd/mmm/list/__snapshots__/` and tests set `MMM_TEST=true` so i18n output is stable.
