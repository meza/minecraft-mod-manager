# `test`

Checks if all configured mods have versions available for a target Minecraft release.

## Behaviour
1. Determine the version to test (argument or `latest`). If `latest` is used, the command queries the official API for the newest release.
2. For each mod in `modlist.json`, fetch metadata for the target version and loader.
3. If every mod has a valid file, the command prints a success message and exits with code `0`.
4. If any mod lacks support, the missing mods are listed and the command exits with code `1`.
5. Supplying the version currently in use results in a no-op with exit code `0`.
6. During execution, per-item status may update in place and output order is not deterministic due to parallel processing; final grouped results are stable.

## Edge Cases
- Invalid or unknown Minecraft versions prompt for a valid version in interactive tty mode and fail fast in unattended or non-tty modes.
- Network failures retrieving version information fail fast in unattended or non-tty modes and instruct the user to supply an explicit version. Interactive tty mode may prompt for an explicit version.
- If configuration files are missing, follow the missing-config gate defined in `docs/interactions/interaction-guidelines.md`.
- If compatibility cannot be determined for one or more mods due to platform errors, report them and exit with code `1`.

## User Interaction
User interaction frames and message shapes are defined in `docs/interactions/flows/test.md`.
