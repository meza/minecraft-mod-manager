# 11. Define CLI error reporting policy

Date: 2025-12-29

## Status

Accepted

## Context

The CLI currently mixes error reporting. Some commands print detailed runtime failures themselves and then return an error that Cobra prints as an extra "Error:" line. Other commands suppress Cobra output for those same errors. This leads to inconsistent output and makes it harder to move toward a full TUI, where runtime errors should be surfaced inside the UI.

We also need clear ownership for input validation errors. Invalid arguments and flags should be reported by Cobra so users get the standard usage feedback and consistent messaging across commands.

## Decision

Commands own runtime error reporting. When a command prints a runtime error to the user, it must return a handled error marker so Cobra does not print a duplicate error line. Cobra continues to report input and usage errors (argument and flag validation), and those should not be handled by command output helpers.

Errors are always printed regardless of `--quiet`. Quiet mode suppresses non-essential output only, not errors.

## Consequences

Output becomes consistent across commands and avoids duplicate "Error:" lines for runtime failures. Command code now needs to explicitly mark handled errors after printing, and tests must cover these paths. Unhandled runtime errors will still surface via Cobra, which provides a safe fallback for unexpected failures.
