# `change`

Switches the configured Minecraft version and reinstalls all mods for the new version.

## Behaviour
1. Verify that the target version is supported by all configured mods (unless `--force` is used). This mirrors the behaviour of the `test` command.
2. Remove all currently installed mod files and update `modlist.json` with the new `gameVersion` value.
3. Run `install` to download compatible versions for the new game version.

## Edge Cases
- Attempting to change to the existing game version exits with code `2`.
- When mods are missing support for the new version, the command fails with code `1` unless `--force` is provided.

## User Interaction
Any errors are logged to the console. With `--force` the version is changed even if some mods fail to install; missing mods are simply skipped.
