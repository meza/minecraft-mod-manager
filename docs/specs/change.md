# `change`

Switches the configured Minecraft version and reinstalls all mods for the new version.

## Behaviour
1. Resolve the target Minecraft version, defaulting to `latest` when no version is provided.
2. Verify that the target version is supported by all configured mods (unless `--force` is used).
3. Download the new mod set without modifying the current setup.
4. If downloads succeed, switch the config and mods folder to the new version.
5. Clean up temporary or backup artifacts. On failure, attempt rollback so no user-visible changes remain.
6. During execution, per-item status may update in place and output order is not deterministic due to parallel processing; final grouped results are stable.

## Edge Cases
- When the configuration is missing, follow the missing-config gate defined in `docs/interactions/interaction-guidelines.md`.
- Attempting to change to the existing game version exits with code `0` and makes no changes.
- When mods are missing support for the new version, the command fails with code `1` unless `--force` is provided.
- When downloads or switching fail, MMM must attempt rollback and emit an actionable error.

## User Interaction
User interaction frames and message shapes are defined in `docs/interactions/flows/change.md`.
