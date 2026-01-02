# `remove`

Deletes one or more mods from both the configuration and the filesystem.

## Behaviour
1. Resolve the provided names or IDs against `modlist-lock.json`, supporting glob patterns against lockfile filenames.
2. When a matching installation exists, delete the file from the mods directory and remove the entry from `modlist-lock.json`.
3. Remove the corresponding mod entry from `modlist.json` and save the updated configuration.
4. When `--dry-run` is used, actions are logged but no files are changed.
5. During execution, per-item status may update in place and output order is not deterministic due to parallel processing; final grouped results are stable.

## Edge Cases
- Mods that do not match any pattern are ignored.
- Missing files are skipped without failing the entire command, but config and lock updates still apply.
- If configuration files are missing, follow the missing-config gate defined in `docs/interactions/interaction-guidelines.md`.
- If a delete fails, the command exits non-zero after reporting the failure.

## User Interaction
User interaction frames and message shapes are defined in `docs/interactions/flows/remove.md`.
