# cmd/mmm/remove

This package implements the `mmm remove` command: remove one or more mods from `modlist.json`, delete any installed
files recorded in `modlist-lock.json`, and update both files accordingly.

The implementation is intentionally small and follows the existing command patterns:

- `cmd/mmm/remove/remove.go`: cobra wiring + `runRemove` implementation
- `cmd/mmm/remove/remove_test.go`: behavior tests (glob resolution, dry-run output, deletion, missing files)

## Glob semantics

Lookup patterns are matched against the configured mod `id` and `name` using Go's built-in `filepath.Match` semantics
against lowercased values. This is intentionally not a full port of Node/minimatch features like brace expansion or
extglobs, per team direction to avoid custom glob implementations.
