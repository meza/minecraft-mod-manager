# Interaction guidelines

This document is the interaction style guide for Minecraft Mod Manager (MMM).
It defines the ethos and the do and do not rules for how MMM behaves across the terminal interaction contexts that exist today.

Scope for this document:
- Non-interactive terminal
- Interactive terminal (tui-lite only)
- `--quiet` flag behavior in both

Out of scope:
- The future full-screen root TUI that launches from `mmm` with no arguments

## Ethos

MMM is a CLI-first tool designed for two realities:
- Automation and repeatability
- Humans operating in terminals under time pressure

The interaction contract must be safe by default, clear under failure, and consistent across commands.

Priority order for trade offs:
1. User success and user safety
2. Accessibility and inclusivity constraints
3. Evidence and validation over preference
4. Reliability and feasibility constraints
5. Polish and novelty

## Model

Terminal mode selection rules are described in `docs/interactions/execution-tiers.md` (historical filename).
Output modes are described in `docs/interactions/patterns/output-modes.md`.
The consolidated, per-command requirements live in `docs/interactions/interaction-consolidation.md`.

MMM has two terminal modes of operation today:
- Non-interactive terminal
- Interactive terminal (tui-lite)

Future:
- Full root-level TUI when you run `mmm` with no args (out of scope)

`--quiet` is a flag that changes what MMM prints.
Each mode MUST handle it according to the quiet rules in this document.

## Universal rules

These rules apply to every command in both terminal modes.

### Safety and reversibility

- Default behavior MUST be non-destructive when user intent is ambiguous.
- Destructive actions MUST be preceded by either:
  - explicit user confirmation, or
  - an explicit opt-in flag such as `--force`
- When MMM refuses to proceed for safety reasons, it MUST say what it detected and what the user can do next.

### No hidden prompts

- MMM MUST NOT prompt when `--non-interactive` is set.
- MMM MUST NOT block on input when stdin or stdout is not a TTY.
  - In these cases, MMM MUST fail fast with an actionable error, unless the command has a defined safe fallback.

### Missing config is a guided recovery in interactive mode

- When config is missing and MMM is interactive, MMM MUST offer to run `init` and then resume the original command.
- When config is missing and MMM is non-interactive, MMM MUST fail fast with an actionable error instructing the user to run `mmm init`.

### Predictable exit codes

- Exit codes MUST be stable and documented for automation.
- Exit code `0` means success.
- Exit code `1` means the command failed and did not accomplish its intended outcome.
- Exit code `2` means a well-defined no-op or "not applicable" state that is not a failure.
  - Such as `test` and `change` when the requested target version equals the configured version.

If a command uses exit code `2`, it MUST be explicitly documented.

### Message quality

- Error messages MUST include:
  - what went wrong
  - the specific input that caused the problem, when applicable
  - the next action the user can take
- MMM MUST NOT require `--debug` for normal comprehension of failures.
- MMM MUST avoid ambiguous phrasing like "failed" without a reason.

### Accessibility

- Every interactive path MUST be keyboard-only.
- Color MUST NOT be the only signal. When color is not available, text and symbols MUST still communicate status.
- Animations and constantly updating output SHOULD be minimized and MUST stop with `--quiet`.

## Terminal modes

### Non-interactive

Non-interactive mode exists for scripts, CI, and deterministic automation.

Do:
- Fail fast when required inputs are missing.
- Print actionable errors.
- Avoid dynamic output and terminal control sequences.
- Require explicit opt-in for destructive actions.

Do not:
- Prompt for missing values.
- Launch Bubble Tea.
- Print progress spinners, interactive tables, or prompts.

### Interactive (tui-lite)

Interactive mode exists for humans using a terminal.
MMM may use:
- simple line prompts (confirmations)
- per-command Bubble Tea flows and wrappers

Do:
- Keep the user oriented with clear step labels and what is being asked.
- Render tui-lite interactions as a transcript:
  - questions on their own lines
  - answers echoed on the same line as the question
  - temporary selection lists collapse into an answered question line
- Keep consistent cancel keys:
  - `ctrl+c` MUST always cancel safely.
  - Where Bubble Tea already provides `q` as a quit key (and shows it in the footer), MMM MUST keep it. Do not remove or rebind it.
- Make defaults explicit in prompt text.
- Provide obvious cancel and back behavior.
- Provide clear outcomes and next steps after completion.

Do not:
- Change the underlying command meaning based on interactivity.
  - Such as if `change` is a no-op for same-version targets, it must be a no-op in every terminal mode.
- Hide destructive behavior behind an unclear prompt.

### Quiet flag

Quiet is a flag for scripts and logs that want minimal noise.
Quiet is silent unless the user needs to take action.

When `--quiet` is set:
- MMM MUST suppress non-essential output.
- MMM MUST still print errors.
- MMM MUST print only actionable results.
  - A result is actionable when it changes what the user should do next.
- Quiet MUST NOT change what input MMM needs.
  - Quiet affects output, not whether MMM prompts in interactive terminal mode.

Do:
- Keep output stable and easy to parse by humans and log processors.
- Prefer one-line summaries when a summary is required.

Do not:
- Print progress animations, spinners, or interactive rendering.
- Hide errors or change exit codes.

#### Actionable output rules

With `--quiet`, MMM prints only what the user needs to action.

Examples:
- `list`: quiet is not meaningful. The list still prints.
- `test --quiet`: silent on success. On failure, print only the mods that block the target version.
- `scan --quiet`: print only unknown or unsure files (things the user must resolve).
- `install --quiet` and `update --quiet`: silent unless there are errors or unmanaged files detected.
- `prune --quiet --force`: silent on success.

Non-essential output includes:
- progress logs during downloads
- repeated "already up to date" lines
- verbose per-file actions that do not change outcomes

## Confirmations

Confirmation patterns are defined in `docs/interactions/patterns/confirmations.md`.

Guidelines:
- Every confirmation prompt MUST include an explicit default like `y/N` or `Y/n`.
- If the default is No, pressing Enter MUST NOT perform the destructive action.
- MMM SHOULD preview what will happen before asking the question.

## Consistency targets

MMM SHOULD feel consistent across commands:
- Same terms for the same concepts (config, lock file, mods folder, unmanaged)
- Same exit code meanings
- Same prompt style and cancel keys
- Same quiet behavior interpretation

If MMM must break consistency for a specific command, the reason MUST be documented and testable.

## Validation plan

To validate that MMM follows these guidelines:
- Run command flows in `--non-interactive`, interactive tty, and `--quiet` variants.
- Capture outputs and exit codes as evidence.
- Treat divergence between terminal modes as a bug unless explicitly documented.

The current black-box discovery plan and capture harness live under:
- `docs/interactions/black-box-discovery.md`
- `docs/interactions/discovery/README.md`
