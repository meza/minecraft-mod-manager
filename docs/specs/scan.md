# `scan`

Searches the mods directory for jar files that are not present in the configuration.

## Behaviour
1. Enumerate files in the mods directory, ignoring entries that end with `.disabled` and respecting patterns from `.mmmignore`.
2. For each file found, query the preferred platform (`-p` or `--prefer`) (default `modrinth`) to identify the matching project.
3. Present the results, grouping recognised files and unknown ones.
4. When executed with `--add`, update `modlist.json` and `modlist-lock.json` to include the discovered mods.
5. File scan and hashing operations are performed in parallel to improve performance and network requests are rate-limited but parallel.
6. The UI/UX matches the node version's `scan` command.

## Edge Cases
- If the platform lookup fails for a file, it is reported as unsure and no changes are made.
- Files already managed in the lock file are skipped.

## User Interaction
Without `--add` and when not running in quiet mode, the user is asked whether to update the configuration after reviewing the results.

## Existing bugs

Despite many attempts of fixing it, scan remains the most flaky command in the Node version.
The following issues track various problems reported by users:

- https://github.com/meza/minecraft-mod-manager/issues/233
- https://github.com/meza/minecraft-mod-manager/issues/261
- https://github.com/meza/minecraft-mod-manager/issues/1005

When porting scan to Go, please ensure to address these issues and add tests to cover them.
The issue most likely stems from bad logic in the node version, so while porting, consider redesigning the flow to be more robust.
