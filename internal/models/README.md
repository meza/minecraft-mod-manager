# internal/models

This package holds the core data types used across the CLI, including the JSON shapes written to disk.

If you change anything in here, assume you are changing the user-facing contract. Keep changes intentional and reviewable.

## What lives here

### Modlist and lockfile shapes

- `ModsJSON` is the shape of `modlist.json`
- `ModInstall` is the shape of each entry in `modlist-lock.json`
- `Mod` is a single mod config in `modlist.json`

### Enums used across commands

- `Platform` (for example `curseforge`, `modrinth`)
- `Loader` (for example `fabric`, `forge`)
- `ReleaseType` (`release`, `beta`, `alpha`)

Helpers like `AllLoaders()` and `AllReleaseTypes()` exist for UI selection flows.

### Platform artifact lookup types

- `FetchOptions` describes how to pick an artifact from a platform.
- `RemoteMod` is the normalized artifact metadata returned by platform-specific packages.

## Related docs

For the user-facing explanation of `modlist.json` and `modlist-lock.json`, see the root [`README.md`](../../README.md).
For authoritative behavior, see the [installation model](../../docs/intent.md#the-installation-model) and [platforms and artifact lookup](../../docs/intent.md#platforms-and-artifact-lookup) in product intent. The [command guide index](../../docs/commands/README.md) routes operator workflows, and [`docs/contributing/interactions/`](../../docs/contributing/interactions/README.md) describes terminal interactions.

## Tests

See the root [verification guidance](../../CONTRIBUTING.md#verification) for required checks and snapshot update instructions.
