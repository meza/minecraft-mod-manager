# Error handling and recovery

This document describes interaction-level error patterns in MMM.
It is not a complete catalog of error types.

## Goals

- Users understand what failed
- Users know what to do next
- Automation receives predictable exit codes

## Handled errors

Some errors are intentionally handled and logged by the command.
In these cases, Cobra error printing is suppressed so output is not duplicated.

Look for:
- `clierrors.IsHandled(err)`
- `cmd.SilenceErrors = true`
- `cmd.SilenceUsage = true`

## Recovery flows

Some commands can recover interactively from expected failures.

Examples:
- `add` uses a Bubble Tea recovery state machine to resolve unknown platforms, not found mods, and missing compatible files.

## Non-interactive contract

In non-interactive environments:
- Do not prompt
- Fail fast when required inputs are missing
- Emit errors that are actionable without interactive context

## Open questions

- Which exit codes are part of the stable automation contract beyond `test`
- Which errors should be treated as handled vs unhandled in each command

