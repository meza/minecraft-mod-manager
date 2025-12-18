# internal/mmmignore

This package implements the current `.mmmignore` semantics used by filesystem-scanning commands.

It intentionally matches the existing behavior found in `cmd/mmm/install`:

- Patterns are read from `.mmmignore` in the config directory.
- Blank lines are ignored.
- A default pattern of `**/*.disabled` is always applied.
- Patterns are evaluated against paths relative to the config directory.

This is not intended to perfectly match Node's glob semantics.

## Public API

- `IgnoredFiles(fs, rootDir) (map[string]bool, error)` returns a set of absolute paths that should be treated as ignored.
- `ListPatterns(fs, rootDir) ([]string, error)` returns the resolved list of patterns (including the default).
- `IsIgnored(rootDir, absolutePath, patterns) bool` checks whether a single absolute path is ignored by the provided patterns.

## Recommended usage

Prefer `ListPatterns` + `IsIgnored` when you already have candidate paths (for example when scanning a mods folder):

```go
patterns, err := mmmignore.ListPatterns(fs, meta.Dir())
if err != nil {
  return err
}

for _, candidatePath := range candidatePaths {
  if mmmignore.IsIgnored(meta.Dir(), candidatePath, patterns) {
    continue
  }
  // process candidatePath
}
```

## Out-of-tree paths

Patterns are rooted at the config directory (`meta.Dir()`). Paths outside that directory are treated as "not ignored".
