# `add`

Adds a mod to the configuration and downloads the corresponding file.

## Behaviour
1. When the configuration is missing, follow the missing-config gate defined in `docs/interactions/interaction-guidelines.md`.
2. When unmanaged jar files are detected in the mods folder, print the unmanaged files notice defined in `docs/interactions/interaction-guidelines.md` and exit without making changes.
3. Fetch metadata for the given `<platform>` and `<id>` using the configured loader and Minecraft version. When `--version` is supplied, that specific version is requested. The optional `--allow-version-fallback` flag allows searching previous Minecraft versions when no file matches the current one.
4. Download the discovered file into the mods directory.
5. Append a mod entry to `modlist.json` and an installation record to `modlist-lock.json`.

## Edge Cases
- Unknown platform is handled by cobra validation.
- If the project ID cannot be found, the user can modify the search or abort (interactive only).
- When no compatible file exists for the selected platform, the user can modify the search or abort (interactive only).
- Download failures allow the user to modify the search or abort (interactive only). Unattended and non-interactive modes fail fast with an error.

## User Interaction
User interaction frames and message shapes are defined in `docs/interactions/flows/add.md`.
