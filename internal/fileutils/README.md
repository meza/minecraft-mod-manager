# internal/fileutils

This package is a small collection of filesystem helpers that work with `afero.Fs`.

It exists so command and HTTP code can be tested with an in-memory filesystem (`afero.NewMemMapFs`) instead of the OS filesystem.

## Public API

- `InitFilesystem(filesystem ...afero.Fs) afero.Fs` (defaults to `afero.NewOsFs()` when no fs is passed)
- `FileExists(path string, filesystem ...afero.Fs) bool`
- `ListFilesInDir(path string, filesystem ...afero.Fs) ([]string, error)` (returns full paths, skips directories)

## Quick start

```go
fs := afero.NewMemMapFs()
_ = afero.WriteFile(fs, "mods/a.jar", []byte("x"), 0644)
paths, err := fileutils.ListFilesInDir("mods", fs)
```

## Tests

Run from the repo root:

```bash
make test
```

