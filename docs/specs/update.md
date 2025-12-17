# `update`

Checks for newer releases of each configured mod and downloads them when available.

## Behaviour
1. Start with an `install` run to ensure the working directory is consistent.
2. For every mod entry, query the remote platform for a newer file matching the configured Minecraft version and loader.
3. When a new release is found, download it, remove the previous file and update the lock file entry.
4. The configuration file is updated to keep mod names in sync.
5. Processes mods in parallel to speed up the operation, but uses the rate limiter configured for the platforms.
6. The UI/UX matches the node version's `update` command.

## Edge Cases
- If a download fails, the previous version remains on disk and the lock file is not altered.
- When a mod is pinned to a specific `version`, it is skipped during updates.
- The command aborts if unmanaged files are detected by the initial `install` phase.

## User Interaction
The command does not ask for input. Progress and potential errors are reported through log output.
