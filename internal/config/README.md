# internal/config

This package owns reading and writing the modlist and lockfile that commands operate on:

- `modlist.json` (the user-managed installation settings and mod configs)
- `modlist-lock.json` (tool-managed records of resolved artifacts)

It is built around dependency injection so command code can be tested without touching the real filesystem or network.

## Quick start

Most commands do some variation of this:

```go
meta := config.NewMetadata(configPath)
cfg, err := config.ReadConfig(fs, meta)
if err != nil {
	// handle missing/invalid config
}

lock, err := config.EnsureLock(fs, meta)
if err != nil {
	// handle lock read/write failures
}
```

## Public API

### Modlist (`modlist.json`)

- `ReadConfig(fs afero.Fs, meta Metadata) (models.ModsJSON, error)`
- `WriteConfig(fs afero.Fs, meta Metadata, cfg models.ModsJSON) error`
- `InitConfig(fs afero.Fs, meta Metadata, minecraftClient httpclient.Doer) (models.ModsJSON, error)`

`InitConfig` creates a minimal modlist when one does not exist yet. It calls `internal/minecraft.GetLatestVersion` to seed `gameVersion`, then writes the file to disk.

### Lockfile (`modlist-lock.json`)

- `EnsureLock(fs afero.Fs, meta Metadata) ([]models.ModInstall, error)` (create empty lock if missing)
- `ReadLockOrEmpty(fs afero.Fs, meta Metadata) ([]models.ModInstall, error)` (return empty lock in memory if missing)
- `ReadLock(fs afero.Fs, meta Metadata) ([]models.ModInstall, error)`
- `WriteLock(fs afero.Fs, meta Metadata, lock []models.ModInstall) error`

### Paths and metadata

`Metadata` keeps the modlist path and provides derived paths:

- `NewMetadata(configPath string) Metadata`
- `Metadata.Dir() string`
- `Metadata.LockPath() string` (same basename as the modlist, with `-lock.json`)
- `Metadata.ModsFolderPath(cfg models.ModsJSON) string` (absolute paths stay absolute; relative paths are relative to the configuration directory)

## Expected errors

`ReadConfig` returns typed errors so commands can decide what to do next:

- `*ConfigFileNotFoundException` when `meta.ConfigPath` does not exist
- `*FileInvalidError` when JSON cannot be unmarshaled

Other failures (read/write permissions, etc) are returned as wrapped `error` values.

## Related docs

For the user-facing shape of these files, see the root [`README.md`](../../README.md).
For authoritative behavior, see the [installation model](../../docs/intent.md#the-installation-model) and [paths and installation boundaries](../../docs/intent.md#paths-and-installation-boundaries) in product intent. The [command guide index](../../docs/commands/README.md) routes operator workflows, and [`docs/interactions/`](../../docs/interactions/README.md) describes terminal interactions.

## Tests

See `CONTRIBUTING.md` for required test/coverage checks and snapshot update instructions.
