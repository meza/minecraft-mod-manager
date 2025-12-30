# Confirmations and cancellation

This document describes confirmation prompts and cancel behavior.

## Confirmation prompts

Some commands use simple line prompts to confirm actions.

Current examples:
- `scan` asks before initializing config and before persisting results when `--add` is not set.

## Defaults

Defaults must be explicit in the prompt string, for example `y/N`.
If the default is No, pressing Enter must not perform the action.

## Cancellation behavior

Cancellation must be explicit and consistent:
- For Bubble Tea flows, `ctrl+c` aborts and `esc` goes back or aborts when there is no previous step.
- For line prompts, EOF should be treated as a safe default, typically No.

## Open questions

- Should all commands use the same confirmation pattern or is command-specific acceptable
- Should confirmations always be available when stdout is redirected but stdin is a TTY

