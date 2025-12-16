# internal/logger

This package provides a very small logger with the two behaviors the CLI needs:

- quiet mode (suppress regular output)
- debug mode (emit extra output)

It is intentionally not a structured logger. The command layer owns what to log and when.

## Public API

- `New(out io.Writer, err io.Writer, quiet bool, debug bool) *Logger`
- `(*Logger).Log(message string, forceShow bool)` (suppressed by `quiet` unless `forceShow` or `debug`)
- `(*Logger).Debug(message string)` (only prints when `debug` is true)
- `(*Logger).Error(message string)` and `(*Logger).Errorf(format string, args ...any)` (always prints to stderr)

## Tests

Run from the repo root:

```bash
make test
```

