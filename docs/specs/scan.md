# `scan`

Searches the mods directory for jar files that are not present in the configuration.

## Behaviour
1. Enumerate files in the mods directory, ignoring entries that end with `.disabled` and respecting patterns from `.mmmignore`.
2. For each file found, query the preferred platform (`-p` or `--prefer`) (default `modrinth`) to identify the matching project.
3. Present the results, grouping recognised files, unknown files, and unsure files. In non-tty mode, results are emitted as per-file lines instead of grouped sections.
4. When executed with `--add`, update `modlist.json` and `modlist-lock.json` to include the recognized matches.
5. File scan and hashing operations are performed in parallel to improve performance and network requests are rate-limited but parallel. Modrinth lookups are per-candidate, while CurseForge lookups batch fingerprints. Both rely on the shared rate limiter to throttle requests.
6. When configuration is missing, follow the missing-config gate defined in `docs/interactions/interaction-guidelines.md`.
7. During execution in tty mode, MMM renders a viewport-based running view in a deterministic, case-insensitive file order and updates lines in place. Items being checked show the spinner, pending hourglass lines appear only before lookups begin, and settled results move into the Recognized, Unknown, or Unsure sections as they are identified.
8. In non-tty mode, MMM prints per-file lines as they settle (completion order) with no grouped sections.
9. When `--add` is set, MMM prints the Added section after per-file lines; in non-tty mode this still includes per-file lines even if every file is recognized.

## Edge Cases
- If the platform lookup fails for a file, it is reported as unsure.
- Files already managed in the lock file are skipped.
- Unknown and unsure files do not block adopting recognised results.

## User Interaction
User interaction frames and message shapes are defined in `docs/interactions/flows/scan.md`.
