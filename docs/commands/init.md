# `init`

Initialises `modlist.json` and prepares the working directory. The command can operate interactively or receive all answers via flags.

## Behaviour
1. Determine the configuration file location. If it already exists and the `--quiet` flag is not used, the user is asked whether to overwrite or provide a new file name.
2. Gather required values (loader, Minecraft version, allowed release types and mods directory) through command line options or interactive prompts.
3. Validate that the chosen mods directory exists and that the supplied Minecraft version is valid. If the game version is omitted, the tool retrieves the latest release from the official API and falls back to a prompt when the API is unavailable.
4. Write the resulting configuration to disk and create an empty `modlist-lock.json`.

## Edge Cases
- Invalid or nonexistent mods folder results in a prompt for a different path.
- Invalid Minecraft versions throw an error and the process aborts.
- When running in `--quiet` mode and the config file already exists, the command exits with an error.

## User Interaction
The user may be prompted to confirm overwriting files, choose values from lists or provide text input. All prompts are skipped when the relevant flags are supplied.
