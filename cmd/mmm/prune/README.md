# cmd/mmm/prune

This package implements the `mmm prune` command: delete unmanaged `.jar` files from the mods folder based on
`modlist-lock.json`, while honoring `.mmmignore` (including the default `**/*.disabled` pattern).

The implementation follows the command conventions used elsewhere in the CLI:

- `cmd/mmm/prune/prune.go`: cobra wiring, prompt handling, deletion logic, and telemetry
- `cmd/mmm/prune/prune_test.go`: behavior tests covering prompts, quiet/force behavior, ignores, and lock errors

## Prompt behavior

Prompts are allowed only when `--non-interactive` is not set and the command is running in a TTY.
When `--non-interactive` is set, prune prints a warning, skips the prompt, and assumes no deletion unless `--force` is also set.
If prompts are disabled because the command is not running in a TTY and `--force` is not set, `mmm prune` prints a warning and exits without deleting files.

## Testing and snapshots

See `CONTRIBUTING.md` for required test/coverage checks and snapshot update instructions.
Snapshots live under `cmd/mmm/prune/__snapshots__/` and tests set `MMM_TEST=true` so i18n output is stable.
