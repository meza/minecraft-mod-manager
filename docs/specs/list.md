# `list`

Displays the mods defined in `modlist.json` and shows whether each one is currently installed.

## Behaviour
1. Load the configuration and lock file.
2. Sort entries alphabetically and print a checkmark for installed mods or a cross for missing or mismatched files. Installed means the lock entry hash matches the local file. When output is not colorized, use V for installed and X for missing.
3. The command reports the total number of mods via telemetry.

## Edge Cases
- If configuration files are missing or invalid, an error is raised.

## User Interaction
Purely informational. No prompts are issued.
