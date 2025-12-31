# Flow: prune

## User goal and success conditions

You want to delete unmanaged files from your mods folder so your installation matches what MMM manages.

Success looks like:
- Unmanaged files are identified using the lock file and `.mmmignore`
- Deletions only happen after explicit confirmation, unless forced
- Deleted files are reported in output

## Entry points

- Command: `mmm prune`
- User guide: `docs/commands/prune.md`
- Behavior spec: `docs/specs/prune.md`

## Primary flow

1. You run `mmm prune`.
2. MMM reads config and lock to determine which files are managed.
3. MMM scans the mods folder for unmanaged jar files, applying `.mmmignore` rules.
4. MMM prints the unmanaged file list.
5. MMM asks for confirmation before deleting, unless `--force` is set.
6. MMM deletes the listed files and prints what was deleted.

## Key alternate paths

- If no unmanaged files are found, MMM prints a message and exits successfully.
- If prompts are required but stdin or stdout are not TTYs, MMM prints a warning and aborts unless `--force` is set.

## Non-interactive behavior

When `--non-interactive` is set and `--force` is not set:
- MMM prints a warning
- MMM lists unmanaged files
- MMM exits without deleting anything

## Feedback and recovery

- The confirmation prompt defaults to no, and pressing Enter should not delete anything.
- Cancellation should not delete anything.

## References in code

- Command implementation: `cmd/mmm/prune/prune.go`
- Prompt gating: `internal/tui/terminal.go`
- Ignore rules: `internal/mmmignore/`
