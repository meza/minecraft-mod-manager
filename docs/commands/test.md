# `test`

Checks if all configured mods have versions available for a target Minecraft release.

## Behaviour
1. Determine the version to test (argument or `latest`). If `latest` is used, the command queries the official API for the newest release.
2. For each mod in `modlist.json`, fetch metadata for the target version and loader.
3. If every mod has a valid file, the command prints a success message and exits with code `0`.
4. If any mod lacks support, the missing mods are listed and the command exits with code `1`.
5. Supplying the version currently in use results in exit code `2`.

## Edge Cases
- Invalid or unknown Minecraft versions cause an error message.
- Network failures retrieving version information prompt the user for the latest version when running interactively.

## User Interaction
Errors and missing mods are printed to the console. No further interaction is required.
