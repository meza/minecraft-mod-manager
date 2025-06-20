# `prune`

Deletes unmanaged files from the mods directory.

## Behaviour
1. Load the configuration and lock file to determine which files are managed.
2. Scan the mods directory and list all files that are not present in `modlist-lock.json`, applying `.mmmignore` rules.
3. Unless `--force` is specified, ask for confirmation before deleting the files.
4. Remove the selected files.

## Edge Cases
- When run in `--quiet` mode without `--force`, the command prints a warning and aborts.
- If no unmanaged files are found, a message is printed and no further action is taken.

## User Interaction
Prompts for confirmation unless the `--force` flag is used. The command outputs which files were deleted.
