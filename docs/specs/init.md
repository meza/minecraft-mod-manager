# `init`

Initialises `modlist.json` and prepares the working directory. The command can operate interactively or receive all answers via flags.

## Behaviour
1. Determine the configuration file location. If it already exists and the command is running interactively, the user is asked whether to overwrite or provide a new file name.
2. Gather required values (loader, Minecraft version, allowed release types and mods directory) through command line options or interactive prompts.
3. Validate that the chosen mods folder exists and that the supplied Minecraft version is valid. If the game version is omitted, the tool retrieves the latest release from the official API.
4. In interactive mode, confirm writing the configuration before proceeding. In unattended or non-tty mode, fail fast when required values are missing or `latest` cannot be resolved.
5. Write the resulting configuration to disk and create an empty `modlist-lock.json`.

## Edge Cases
- Invalid or nonexistent mods folder results in a prompt for a different path in interactive mode.
- Invalid Minecraft versions keep the user in the version input step (interactive) and fail fast in unattended mode.
- When running in `--unattended` or non-tty mode and the config file already exists, the command exits with an error unless `--force` is supplied.
- Missing required values in unattended mode exits with an error.
- `latest` resolution failures in unattended or non-tty mode exit with an error.
- Mods folder does not exist in unattended or non-tty mode exits with an error.

## User Interaction
User interaction frames and message shapes are defined in `docs/interactions/flows/init.md`.
