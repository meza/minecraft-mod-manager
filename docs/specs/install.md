# `install`

Downloads all mods listed in `modlist.json` according to the lock file.

## Behaviour
1. Load the configuration and existing `modlist-lock.json` records.
2. For each configured mod:
   - If a matching installation exists, verify the local file's hash and redownload when it differs or is missing.
   - If no installation exists, fetch remote metadata and download the file.
3. When lock entries are missing for configured mods, add them after download. Otherwise, leave existing lock entries intact.
4. When a mod author has changed the mod name, update the name in `modlist.json` and the lock file entry for that mod.

## Edge Cases
- Hash mismatches trigger re-downloads to ensure integrity.
- Network or download failures surface as errors.
- Failure to write lock updates surfaces as an error.

## User Interaction
User interaction frames and message shapes are defined in `docs/interactions/flows/install.md`.
