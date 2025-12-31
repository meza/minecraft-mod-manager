# `prune`

Deletes unmanaged files from the mods directory.

## Behaviour
1. Load the configuration and lock file to determine which files are managed.
2. Scan the mods directory and list all files that are not present in `modlist-lock.json`, applying `.mmmignore` rules.
3. Unless `--force` is specified, ask for confirmation before deleting the files. In `--non-interactive` mode, print a warning, skip the prompt, and assume no deletion.
4. Remove the selected files.

## Edge Cases
- When run in `--non-interactive` mode without `--force`, the command prints a warning, lists unmanaged files, and exits without deleting them.
- When prompts are required but stdin or stdout are not TTYs, the command prints a warning and aborts unless `--force` is set.
- If no unmanaged files are found, a message is printed and no further action is taken.

## User Interaction
Prompts for confirmation unless the `--force` flag is used. In `--non-interactive` mode, the prompt is skipped and no deletion occurs without `--force`. The command outputs which files were deleted.
