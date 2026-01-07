# cmd/mmm/prune

This package implements the `mmm prune` command: delete unmanaged `.jar` files from the mods folder based on
`modlist-lock.json`, while honoring `.mmmignore` (including the default `**/*.disabled` pattern).

The implementation follows the command conventions used elsewhere in the CLI:

- `cmd/mmm/prune/prune.go`: cobra wiring, prompt handling, deletion logic, and telemetry
- `cmd/mmm/prune/prune_test.go`: behavior tests covering prompts, quiet/force behavior, ignores, and lock errors

## Prompt behavior

Prompts are allowed only in interactive mode. Interactive mode requires both stdin and stdout to be TTYs and `--unattended` to be unset.

When prompts are not allowed and `--force` is not set, prune prints the unmanaged list and refuses to delete files. This applies to both `--unattended` and non-tty runs.
When `--force` is set, prune skips prompting and deletes unmanaged files immediately.

When the config file is missing and prompts are allowed, prune offers to run `mmm init` and then resumes. In unattended or non-tty contexts, it prints the missing config error and exits.

## Testing and snapshots

See `CONTRIBUTING.md` for required test/coverage checks and snapshot update instructions.
Snapshots live under `cmd/mmm/prune/__snapshots__/` and tests set `MMM_TEST=true` so i18n output is stable.
