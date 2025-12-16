# internal/config

This package owns reading and writing the two config files that commands operate on:

- `modlist.json` (user-managed intent)
- `modlist-lock.json` (tool-managed resolved files)

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

### Config file (`modlist.json`)

- `ReadConfig(fs afero.Fs, meta Metadata) (models.ModsJson, error)`
- `WriteConfig(fs afero.Fs, meta Metadata, cfg models.ModsJson) error`
- `InitConfig(fs afero.Fs, meta Metadata, minecraftClient httpClient.Doer) (models.ModsJson, error)`

`InitConfig` creates a minimal config file when one does not exist yet. It calls `internal/minecraft.GetLatestVersion` to seed `gameVersion`, then writes the file to disk.

### Lock file (`modlist-lock.json`)

- `EnsureLock(fs afero.Fs, meta Metadata) ([]models.ModInstall, error)` (create empty lock if missing)
- `ReadLock(fs afero.Fs, meta Metadata) ([]models.ModInstall, error)`
- `WriteLock(fs afero.Fs, meta Metadata, lock []models.ModInstall) error`

### Paths and metadata

`Metadata` keeps the config path and provides derived paths:

- `NewMetadata(configPath string) Metadata`
- `Metadata.Dir() string`
- `Metadata.LockPath() string` (same basename as config, with `-lock.json`)
- `Metadata.ModsFolderPath(cfg models.ModsJson) string` (absolute paths stay absolute; relative paths are relative to the config directory)

## Expected errors

`ReadConfig` returns typed errors so commands can decide what to do next:

- `*ConfigFileNotFoundException` when `meta.ConfigPath` does not exist
- `*ConfigFileInvalidError` when JSON cannot be unmarshaled

Other failures (read/write permissions, etc) are returned as wrapped `error` values.

## Related docs

For the user-facing shape of these files, see the root `README.md` and `docs/requirements-go-port.md`.

## Tests

Run from the repo root:

```bash
make test
```

