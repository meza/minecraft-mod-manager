# `update`

Checks for newer releases of each configured mod and downloads them when available.

## Behaviour
1. When mods are configured, start with an `install` run to ensure the working directory is consistent, using the install output as part of UPDATE-01 in the update flow. In non-tty mode, print `Installing potentially missing mods:` before the install transcript.
2. For every mod entry, query the remote platform for a newer file matching the configured Minecraft version and loader.
3. When a new release is found, download it, remove the previous file and update the lock file entry.
4. The configuration file is updated to keep mod names in sync.
5. Continue updating other mods even if a single mod fails to update.
6. Report results grouped by outcome (already up to date, updated, skipped, failed).
7. Processes mods in parallel to speed up the operation, but uses the rate limiter configured for the platforms.
8. During execution, per-item status may update in place and output order is deterministic in tty; final grouped results are stable.

## Edge Cases
- If a download fails, the previous version remains on disk and the lock file is not altered.
- When a mod is pinned to a specific `version`, it is skipped during updates.
- The command aborts if unmanaged files are detected by the preflight check (including the initial `install` phase), after printing the unmanaged files notice defined in `docs/interactions/interaction-guidelines.md`.
- If no mods are configured and no unmanaged files are detected, the command exits successfully and reports that state.
- If no updates are available, the command exits successfully with only the up-to-date (and skipped) results.
- If writing lock updates fails, the command exits non-zero with an actionable error that matches UPDATE-ERR-WRITE-LOCK in the update flow.
- If writing config updates fails, the command exits non-zero with an actionable error.

## User Interaction
User interaction frames and message shapes are defined in `docs/interactions/flows/update.md`.
