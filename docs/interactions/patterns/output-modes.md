# Output modes

This document describes output-level interaction modes used across commands.

## Quiet flag

When `--quiet` is set:
- Suppress non-actionable output
- Errors still print
- Actionable results still print

Quiet is intended for scripts and logs that want minimal noise.
Quiet can be used with either non-interactive or interactive terminal (tui-lite).

Actionable means one of:
- The command failed
- The user must take a follow up action
- The command exists to display information as the primary output

Exception:
- `list --quiet` still prints the list. Quiet is not meaningful for list.

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
