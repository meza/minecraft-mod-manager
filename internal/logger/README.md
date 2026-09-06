# internal/logger

This package provides a small diagnostic logger. It is intentionally unstructured and is not used for user-facing command output.

Debug logging is enabled by the `-d` flag. User-facing output lives in `internal/output`.

## Public API

- `New(out io.Writer, err io.Writer, quiet bool, debug bool) *Logger`
- `(*Logger).Log(message string, visibility LogVisibility)`
- `(*Logger).Debug(message string)`
- `(*Logger).Error(message string)` and `(*Logger).Errorf(format string, args ...any)`

All logger methods return write errors to callers, except for broken pipes which are treated as non-fatal.

## Tests

See the root [verification guidance](../../CONTRIBUTING.md#verification) for required checks and snapshot update instructions.
