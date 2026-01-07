# `prune`

Deletes unmanaged files from the mods directory.

## Behaviour
1. Load the configuration and lock file to determine which files are managed.
2. Scan the mods directory and list all files that are not present in `modlist-lock.json`, applying `.mmmignore` rules.
3. Unless `--force` is specified, ask for confirmation before deleting the files. In `--unattended` mode and non-tty mode, do not prompt and require `--force` to delete.
4. Remove the selected files.
5. Output is ordered deterministically by filename; per-item status is rendered consistently across execution contexts.

## Edge Cases
- When run in `--unattended` mode without `--force`, the command prints a warning, lists unmanaged files, and exits without deleting them.
- If no unmanaged files are found, a message is printed and no further action is taken.
- If the lock file is missing, the command errors and instructs the user to run `mmm install`.
- If deletion fails, the command prints an actionable error.

## User Interaction
User interaction frames and message shapes are defined in `docs/interactions/flows/prune.md`.
