# `scan`

Searches the mods directory for jar files that are not present in the configuration.

## Behaviour
1. Enumerate files in the mods directory, ignoring entries that end with `.disabled` and respecting patterns from `.mmmignore`.
2. For each file found, query the preferred platform (default `modrinth`) to identify the matching project.
3. Present the results, grouping recognised files and unknown ones.
4. When executed with `--add`, update `modlist.json` and `modlist-lock.json` to include the discovered mods.

## Edge Cases
- If the platform lookup fails for a file, it is reported as unsure and no changes are made.
- Files already managed in the lock file are skipped.

## User Interaction
Without `--add` and when not running in quiet mode, the user is asked whether to update the configuration after reviewing the results.
