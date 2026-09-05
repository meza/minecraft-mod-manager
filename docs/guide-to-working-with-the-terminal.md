# Guide to working with the terminal

This is a developer guide for implementing and maintaining MMM terminal interactions.

The user-visible behavior contract lives in `docs/interactions/interaction-guidelines.md`.
This document focuses on implementation practices that keep behavior consistent and testable across execution contexts.

The `cmd/mmm/init` package is the reference implementation for the patterns described here.

## Execution contexts

MMM has multiple execution contexts with different interaction constraints.
Definitions and requirements are owned by `docs/interactions/interaction-guidelines.md`.

When you are implementing a command, ensure behavior is correct in:
- non-interactive terminal
- unattended terminal
- interactive terminal

See:
- `docs/interactions/interaction-guidelines.md#execution-contexts`
- `docs/interactions/interaction-guidelines.md#non-interactive-terminal`
- `docs/interactions/interaction-guidelines.md#unattended-terminal-tui-lite`
- `docs/interactions/interaction-guidelines.md#interactive-terminal-tui-lite`

## Decide when prompting is allowed

Use execution context detection to decide whether the command can run an interactive flow.

Use `internal/view` as the shared helper for view capabilities and Bubble Tea program options:
- `internal/view.SupportsPrompting(in, out)` to decide whether prompts are allowed
- `internal/view.ProgramOptions(in, out)` to ensure Bubble Tea does not emit terminal control sequences when stdout is not a TTY

See `cmd/mmm/init/run.go` for the execution mode selection pattern.

This section implements the interaction contract rules that ban prompting outside interactive contexts and require safe degradation:
- `docs/interactions/interaction-guidelines.md#no-hidden-prompts`
- `docs/interactions/interaction-guidelines.md#unattended-and-non-interactive-contract`
- `docs/interactions/interaction-guidelines.md#output-streams`

## Build interactive terminal flows as command apps

When an interactive flow is allowed, build one Bubble Tea app per command and make it purpose-built for that command.
Avoid a reusable prompt framework.
Model the flow as a finite state machine (FSM) and lock reviewed product behavior down with observable E2E scenarios.

These patterns exist to keep terminal interaction consistent with the transcript-first requirements:
- `docs/interactions/interaction-guidelines.md#rendering-model`
- `docs/interactions/interaction-guidelines.md#tui-lite-only`
- `docs/interactions/interaction-guidelines.md#bubble-tea-only`

### Keep the model in the command package

- Keep the Bubble Tea model in the command package (example: `cmd/mmm/init/interactive_flow.go`).
- Avoid generalized abstractions (generic wizards, generic prompts, generic screens). This project is an application, not a terminal UI library.
- Keep interaction concerns (state, layout, key handling) separate from business logic (API calls, config, file system changes) by injecting the command functions the model needs to call.

See `docs/interactions/interaction-guidelines.md#cobra-validation-vs-runtime-recovery` for how argv validation and interactive recovery flows are expected to relate.

### Model the flow as an explicit FSM

- Define an enum-like state type (example: `type state int`) and a constant per step.
- Centralize transitions so each state owns its prompt, defaults, and bubble configuration.
- Keep state transitions explicit and easy to exercise through user actions.

The `cmd/mmm/init/interactive_flow.go` model uses `nextMissingState(...)` as the single place that decides what step comes next.

### Compose prompt models

Compose a single command-level model from smaller prompt models.
Each prompt model owns its own input behavior and returns a typed `tea.Msg` when the user confirms a choice.

This keeps each question small and testable, and it keeps the command-level FSM focused on:
- which step is current
- how to apply a selected value to the result
- what the next missing step is

See the model files and `cmd/mmm/init/confirm_prompt.go` under `cmd/mmm/init` for concrete examples.

See `docs/interactions/interaction-guidelines.md#selection-list-rendering` for the selection list collapse and transcript persistence requirements.

### Cancellation behavior

Follow the behavior contract in `docs/interactions/interaction-guidelines.md`.

The `cmd/mmm/init` interactive flow uses these defaults:
- `ctrl+c` cancels the flow safely (handled at the command model level)
- `esc` cancels the current prompt and exits the flow (handled by prompt models)

If a flow needs back navigation, implement it explicitly and cover the reviewed behavior with a product scenario.

See `docs/interactions/interaction-guidelines.md#cancel-behavior`.

## Render as a transcript

Interactive output should remain readable as a line-oriented transcript.

The `cmd/mmm/init/interactive_flow.go` model shows a durable approach:
- build sections for each step
- render completed steps as answered prompt lines
- render only the current prompt as an interactive control
- avoid clearing the screen or hiding previous lines

See:
- `docs/interactions/interaction-guidelines.md#interactive-terminal-tui-lite`
- `docs/interactions/interaction-guidelines.md#tui-lite-only`
- `docs/interactions/interaction-guidelines.md#selection-list-rendering`

## Language dependent option initials

Some prompts accept a short token (often a single character) that depends on locale.
Do not hardcode English tokens like `y/N`.

The `cmd/mmm/init/confirm_prompt.go` model shows the preferred pattern:
- each option has a stable meaning in code, plus localized `label` and `short` values from i18n
- defaults are owned by code, not translations
- the parser accepts both localized short token and localized full label
- invalid input is handled by re-prompting in interactive context, and by safe fallback in non-interactive contexts

See `docs/interactions/interaction-guidelines.md#language-dependent-prompts-option-initials`.

## Terminal behavior tests

Godog scenarios driven through tui-test are the product regression harness for terminal UX. They launch the native E2E-tagged MMM binary and observe i18n keys, terminal state, exit status, and filesystem effects.

Use in-process model tests for domain state transitions and rare failure paths that are not product conversations. Do not treat existing model or output snapshots as authoritative product requirements.

Use a tui-test snapshot only when the reviewed requirement depends on complete layout or styling. Do not recreate project-owned PTY, emulation, normalization, polling, or input helpers.

See `docs/testing/terminal-harness.md` for the E2E adapter boundary and `docs/testing/bdd.md` for scenario design.

See `docs/interactions/interaction-guidelines.md#validation-plan` for how to validate behavior across execution contexts.

## TTY behavior

Use `internal/view.ProgramOptions(in, out)` when you construct a Bubble Tea program so it disables the renderer when stdin or stdout are not TTYs.
For tests that need deterministic behavior across platforms, override terminal detection via `view.SetIsTerminalFuncForTesting(...)` and restore it afterwards.

This is the implementation of:
- `docs/interactions/interaction-guidelines.md#non-interactive-terminal`
- `docs/interactions/interaction-guidelines.md#output-streams`
