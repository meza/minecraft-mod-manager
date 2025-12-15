# `remove`

Deletes one or more mods from both the configuration and the filesystem.

## Behaviour
1. Resolve the provided names or IDs against `modlist.json`, supporting glob patterns for partial matches.
2. When a matching installation exists, delete the file from the mods directory and remove the entry from `modlist-lock.json`.
3. Remove the corresponding mod entry from `modlist.json` and save the updated configuration.
4. When `--dry-run` is used, actions are logged but no files are changed.

## Edge Cases
- Mods that do not match any pattern are ignored.
- Missing files are skipped without failing the entire command.

## User Interaction
The command prints the name of each mod removed. With `--dry-run` it reports what would be removed without performing any deletions.
