# `install`

Downloads all mods listed in `modlist.json` according to the lock file.

## Behaviour
1. Load the configuration and existing `modlist-lock.json` records.
2. Scan the mods directory for unmanaged files; unresolved matches produce an error suggesting a manual `scan` run.
3. For each configured mod:
   - If a matching installation exists, verify the local file's hash and redownload when it differs or is missing.
   - If no installation exists, fetch remote metadata and download the file.
4. Update `modlist-lock.json` and persist any new names in `modlist.json`.

## Edge Cases
- Hash mismatches trigger re-downloads to ensure integrity.
- Unmanaged files halt the process until the user resolves them with the `scan` command.
- Network or download failures surface as errors.

## User Interaction
The command is mostly non‑interactive, but error messages describe unresolved files and how to address them.
