# Output modes

This document describes output-level interaction modes used across commands.

## Quiet mode

When `--quiet` is set:
- Suppress non-essential output
- Errors and required results still print

Quiet mode is intended for scripts that want deterministic output.

## Debug mode

When `--debug` is set:
- Emit additional diagnostic output

Debug output should not be required to understand normal failures.

## Color and icons

Colorized output is used only when output is a terminal.
When not colorized, commands should still communicate state clearly using plain text.

## Performance logs and telemetry

Some runs can produce performance logs and emit best-effort telemetry.
These are not interactive features but they affect what users see.

See:
- `README.md` for `--perf` and telemetry

