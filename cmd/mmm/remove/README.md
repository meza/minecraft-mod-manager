# cmd/mmm/remove

This package implements the `mmm remove` command: remove one or more mod configs from `modlist.json`, delete the local
files named by their lock entries in `modlist-lock.json`, and update both files accordingly.

The implementation is intentionally small and follows the existing command patterns:

- `cmd/mmm/remove/remove.go`: cobra wiring + `runRemove` implementation
- `cmd/mmm/remove/remove_test.go`: behavior tests (glob resolution, confirmation, deletion, missing files)

## Glob semantics

Mod matching patterns are matched against each mod config's `id` and `name` using Go's built-in `filepath.Match` semantics
against lowercased values. This is intentionally not a full port of Node/minimatch features like brace expansion or
extglobs, per team direction to avoid custom glob implementations.
