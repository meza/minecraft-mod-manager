# internal/output

This package provides user-facing output helpers for command output. It owns quiet-mode behavior and ensures write errors are handled explicitly.

## Public API

- `New(out io.Writer, err io.Writer, quiet bool) *Output`
- `(*Output).Log(message string, visibility LogVisibility)`
- `(*Output).Error(message string)` and `(*Output).Errorf(format string, args ...any)`

## Behavior

- `Log` respects quiet mode unless `LogForce` is used.
- `Error` and `Errorf` always write to stderr.
- Write failures are returned to callers, except for broken pipes which are treated as non-fatal.

## Tests

See `CONTRIBUTING.md` for required test/coverage checks and snapshot update instructions.
