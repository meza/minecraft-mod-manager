# `scan`

Searches the mods directory for jar files that are not present in the configuration.

## Behaviour
1. Enumerate files in the mods directory, ignoring entries that end with `.disabled` and respecting patterns from `.mmmignore`.
2. For each file found, query the preferred platform (`-p` or `--prefer`) (default `modrinth`) to identify the matching project.
3. Present the results, grouping recognised files, unknown files, and unsure files.
4. When executed with `--add`, update `modlist.json` and `modlist-lock.json` to include the discovered mods.
5. File scan and hashing operations are performed in parallel to improve performance and network requests are rate-limited but parallel.
6. When configuration is missing, follow the missing-config gate defined in `docs/interactions/interaction-guidelines.md`.
7. During execution, per-item status may update in place and output order is not deterministic due to parallel processing; final grouped results are stable.

## Edge Cases
- If the platform lookup fails for a file, it is reported as unsure and no changes are made.
- Files already managed in the lock file are skipped.
- Unknown and unsure files do not block adopting recognised results.

## User Interaction
User interaction frames and message shapes are defined in `docs/interactions/flows/scan.md`.
