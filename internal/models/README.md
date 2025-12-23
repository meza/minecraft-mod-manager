# internal/models

This package holds the core data types used across the CLI, including the JSON shapes written to disk.

If you change anything in here, assume you are changing the user-facing contract. Keep changes intentional and reviewable.

## What lives here

### Config and lock shapes

- `ModsJSON` is the shape of `modlist.json`
- `ModInstall` is the shape of each entry in `modlist-lock.json`
- `Mod` is a single configured mod entry in `modlist.json`

### Enums used across commands

- `Platform` (for example `curseforge`, `modrinth`)
- `Loader` (for example `fabric`, `forge`)
- `ReleaseType` (`release`, `beta`, `alpha`)

Helpers like `AllLoaders()` and `AllReleaseTypes()` exist for UI selection flows.

## Related docs

For the user-facing explanation of `modlist.json` and `modlist-lock.json`, see the root `README.md` and `docs/requirements-go-port.md`.

## Tests

See `CONTRIBUTING.md` for required test/coverage checks and snapshot update instructions.
