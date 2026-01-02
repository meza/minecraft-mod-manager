# `list`

Displays the mods defined in `modlist.json` and shows whether each one is currently installed.

## Behaviour
1. Load the configuration and lock file, following the missing-config gate defined in `docs/interactions/interaction-guidelines.md`.
2. Sort entries alphabetically and render installed vs missing states based on the lock hash matching the local file.
3. If unmanaged jar files are detected, print the unmanaged files notice after the list.
4. The command reports the total number of mods via telemetry.

## Edge Cases
- If configuration files are missing or invalid, an error is raised.
- If no mods are configured, render the empty-list message.
- If a mod has a hash mismatch, mark it missing and suggest running `mmm install`.

## User Interaction
User interaction frames and message shapes are defined in `docs/interactions/flows/list.md`.
