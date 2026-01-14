# Modlist Fixture Helpers

This package provides a small, test-focused helper for creating temporary `modlist.json` and `modlist-lock.json` paths with cleanup handled for you. It does not write modlist contents unless your test calls the helper methods.

## Quick Start (in-memory filesystem)

Use the default in-memory filesystem for fast unit tests:

```go
fixture, err := modlistfixture.New(t)
require.NoError(t, err)

cfg := models.ModsJSON{
	Loader:      models.FABRIC,
	GameVersion: "1.20.1",
	ModsFolder:  "mods",
	Mods:        []models.Mod{},
}
require.NoError(t, fixture.WriteConfig(context.Background(), cfg))
require.NoError(t, fixture.WriteLock(context.Background(), nil))
_, err = fixture.EnsureModsFolder(cfg)
require.NoError(t, err)
```

## Real Filesystem (black-box tests)

Use the OS filesystem when your test needs real paths (for example, end-to-end or CLI black-box tests). This registers a SIGINT/SIGTERM cleanup hook via `internal/lifecycle`.

```go
fixture, err := modlistfixture.New(t, modlistfixture.WithOSFilesystem())
require.NoError(t, err)

configPath := fixture.ConfigPath
lockPath := fixture.LockPath
```

## Options

### WithFS

Use a custom `afero.Fs`. This disables lifecycle cleanup registration unless you also pass `WithOSFilesystem`.

### WithOSFilesystem

Uses `afero.NewOsFs()` and registers cleanup on SIGINT/SIGTERM. This is the only mode that installs a lifecycle hook.

### WithBaseDir

Sets the parent directory for the temp fixture location. When empty, the OS or `afero.TempDir` default is used.

## Helper Methods

- `WriteConfig(ctx, cfg)` writes `modlist.json` for your test.
- `WriteLock(ctx, lock)` writes `modlist-lock.json` for your test.
- `EnsureModsFolder(cfg)` creates the mods folder based on `cfg`.

## Cleanup Behavior

`New` registers `t.Cleanup` for you. Calling `fixture.Cleanup()` is safe and idempotent if you want to control cleanup earlier in your test.
