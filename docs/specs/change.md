# `change`

Switches the configured Minecraft version and reinstalls all mods for the new version.

## Behaviour
1. When unmanaged jar files are detected in the mods folder, print the unmanaged files notice defined in `docs/interactions/interaction-guidelines.md` and exit without making changes.
2. Resolve the target Minecraft version, defaulting to `latest` when no version is provided.
3. Start per-mod compatibility checks for the target version unless `--force` is used.
   - When `--force` is set, choose an incompatible-mod policy:
     - `--keep-config` (default): keep mods in config and remove incompatible jars.
     - `--prune-config`: remove mods from config and remove incompatible jars.
     - `--disable-skipped`: keep mods in config and rename incompatible jars to `*.jar.disabled`.
   - In interactive mode, prompt for the policy when none is provided.
   - In unattended or non-interactive mode, default to `--keep-config` when none is provided.
4. As mods are confirmed compatible, resolve and download them in parallel into a staging area under `mods/.mmm-staging`, without modifying the current setup.
5. If any compatibility checks fail and `--force` is not set, cancel outstanding downloads and roll back staging artifacts, but continue remaining compatibility checks and only exit non-zero after all checks complete.
6. If all required downloads succeed, move the currently installed jars into `mods/.mmm-staging/backup`.
7. Move staged jars into the mods folder, update `modlist-lock.json`, and update `modlist.json` with the new `gameVersion`.
8. Clean up staging and backup artifacts. On failure during switching, attempt rollback so no user-visible changes remain.

During execution, per-item status may update in place. Mod list ordering follows `docs/interactions/interaction-guidelines.md`.

## Edge Cases
- When the configuration is missing, follow the missing-config gate defined in `docs/interactions/interaction-guidelines.md`.
- Attempting to change to the existing game version exits with code `0` and makes no changes.
- When mods are missing support for the new version, the command fails with code `1` unless `--force` is provided.
- When downloads or switching fail, MMM must attempt rollback and emit an actionable error.

## User Interaction
User interaction frames and message shapes are defined in `docs/interactions/flows/change.md`.
