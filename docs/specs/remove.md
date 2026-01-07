# `remove`

Deletes one or more mods from both the configuration and the filesystem.

## Behaviour
1. Resolve the provided names or IDs against `modlist-lock.json`, supporting glob patterns against mod IDs and names.
2. For any matching lock entry, delete its jar from the mods directory, remove the matching mod entry from `modlist.json` by ID, then remove the lock entry.
3. For any matching config entry without a lock entry, remove the entry from `modlist.json` by ID or name (even if the same lookup also matched lock entries).
4. When `--dry-run` is used, actions are logged but no files are changed.
5. During execution, tty output updates items in place with a stable list order; non-tty output is a plain transcript in completion order.

## Edge Cases
- Mods that do not match any pattern are ignored.
- Missing files are skipped without failing the entire command, but config and lock updates still apply.
- If configuration files are missing, follow the missing-config gate defined in `docs/interactions/interaction-guidelines.md`.
- If a delete fails, the command exits non-zero after reporting the failure.

## User Interaction
User interaction frames and message shapes are defined in `docs/interactions/flows/remove.md`.
