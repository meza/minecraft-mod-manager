# `add`

Adds a mod to the configuration and downloads the corresponding file.

## Behaviour
1. Ensure `modlist.json` and `modlist-lock.json` exist, creating them when needed.
2. Fetch metadata for the given `<platform>` and `<id>` using the configured loader and Minecraft version. When `--version` is supplied, that specific version is requested. The optional `--allow-version-fallback` flag allows searching previous Minecraft versions when no file matches the current one.
3. Download the discovered file into the mods directory.
4. Append a mod entry to `modlist.json` and an installation record to `modlist-lock.json`.

## Edge Cases
- Unknown platform prompts the user to pick a valid platform.
- If the project ID cannot be found, the user is asked to modify the search or abort.
- When no compatible file exists for the selected platform, the user is offered the choice to try the alternate platform and enter a different ID.
- Download failures terminate the command with an error.

## User Interaction
Depending on flags and failures, the command may prompt the user to adjust the platform or project ID, or to confirm retrying on a different platform. Quiet mode skips prompts and fails immediately on errors.
