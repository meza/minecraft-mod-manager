# 10. Define unattended and quiet CLI flags

Date: 2025-12-29

## Status

Accepted

## Context

The CLI currently mixes three concerns: whether prompts are allowed, whether stdin or stdout are terminals, and whether the user wants reduced output. This makes automation behavior hard to predict because the same flag can be used for both prompt control and output suppression. We need a clear, stable contract that keeps scripts from hanging, preserves useful output for humans, and avoids conflating TTY detection with output policy.

Automation use cases also require a strict fail-fast behavior: when input cannot be prompted for, missing non-default values must produce an immediate error rather than a prompt or a hang.

## Decision

We will introduce a `--unattended` flag that controls prompt policy. When `--unattended` is set, commands must not prompt. If any required input that does not have a default is missing, the command must exit with an error. Inputs with defined defaults may still use those defaults.

We will keep `--quiet` as an output policy only. When `--quiet` is set, commands suppress non-essential output, but still print errors and required result lines. The exact required result lines are command-specific, and they remain visible so automation can still rely on the output when necessary.

TTY detection only controls whether a renderer or interactive UI is used, not whether prompts are allowed. Non-TTY mode is treated as unattended by default, so it follows the same "no prompts and fail fast when required input is missing" rule.

Examples:
- `mmm --quiet list` still prints the list results (required output) but suppresses non-essential messages.
- `mmm --unattended init -l fabric -g 1.21.1 -m ./mods` runs without prompts and uses defaults for any optional values.
The command fails fast if a required value such as `--loader` is omitted.
- `mmm --unattended --quiet test 1.20.4` runs without prompts and limits output to errors and required results.

## Consequences

Automation becomes predictable because prompt policy is explicit and non-TTY runs fail fast when required inputs are missing. Output behavior becomes independently configurable, allowing users to suppress chatter without disabling prompts when they are available.

Commands and shared helpers will need to be updated to follow this separation. Tests and documentation must be updated to reflect the new `--unattended` flag and the fact that `--quiet` no longer controls prompting.
