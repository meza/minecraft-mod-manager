# Interaction guidelines

This document is the interaction style guide for Minecraft Mod Manager (MMM).
It defines the ethos and the do and do not rules for how MMM behaves across the terminal interaction contexts that exist today.

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

### Rendering model

MMM interactions are transcript-first across non-interactive, unattended, and interactive terminal contexts.
They MUST look and feel like traditional terminal input and output, with small enhancements for interactive selection and recovery.

### tui-lite mode

tui-lite mode refers to the fact that MMM uses Bubble Tea to render prompts and transient UI while preserving a normal line oriented transcript.
This does not mean MMM is doing stdin/stdout prompting.
It means the interaction behaves like it.

### Modes of operation

Per-command flow documents live in `docs/interactions/flows`.

MMM has three terminal execution contexts today:
- Non-interactive terminal
- Unattended terminal (tui-lite)
- Interactive terminal (tui-lite)

Definitions and rules for these contexts are defined in `## Execution contexts`.

Future:
- Full root-level TUI when you run `mmm` with no args (out of scope)

`--quiet` is a flag that changes what MMM prints.
Each mode MUST handle it according to the quiet rules in this document.

## Execution contexts

### Non-interactive terminal

Non-interactive mode exists for transcript-only output in environments where stdin or stdout is not a TTY.
In non-interactive mode, MMM MUST NOT show interactive elements and MUST avoid terminal control sequences.

Do:
- Fail fast when required inputs are missing.
- Print actionable errors.
- Avoid dynamic output and terminal control sequences.
- Require explicit opt-in for destructive actions.

Do not:
- Prompt for missing values.
- Print progress spinners, interactive tables, or prompts.

### Unattended terminal (tui-lite)

Unattended mode exists for scripts, CI, and deterministic automation.
Unattended refers to prompting behavior, not whether the terminal is interactive.
In unattended mode, MMM MUST NOT prompt and MUST fail fast when required inputs are missing.

Do:
- Fail fast when required inputs are missing.
- Print actionable errors.
- Require explicit opt-in for destructive actions.

Do not:
- Prompt for missing values.

### Interactive terminal (tui-lite)

Interactive mode exists for humans using a terminal.
MMM uses per-command Bubble Tea flows and wrappers that render a line oriented transcript.

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
  - Such as if `change` is a no-op for same-version targets, it must be a no-op in every execution context.
- Hide destructive behavior behind an unclear prompt.

## Universal rules

These rules apply to every command in all execution contexts.

### Safety and reversibility

### Cobra validation vs runtime recovery

CLI parsing and validation errors are handled by Cobra before command execution.
Runtime resolution failures are handled by MMM, and interactive mode MAY offer recovery flows that change earlier choices (including choices originally provided via argv).

- Default behavior MUST be non-destructive when user intent is ambiguous.
- Destructive actions MUST be preceded by either:
  - explicit user confirmation, or
  - an explicit opt-in flag such as `--force`
- When MMM refuses to proceed for safety reasons, it MUST say what it detected and what the user can do next.

### No hidden prompts

- MMM MUST NOT prompt when `--unattended` is set.
- MMM MUST NOT block on input when stdin or stdout is not a TTY.
  - In these cases, MMM MUST fail fast with an actionable error, unless the command has a defined safe fallback.

### Output streams

MMM MUST NOT use output stream routing as a UX mechanism.
Bubble Tea output MUST render to stdout.

When stdout is not a TTY, MMM MUST degrade output in an idiomatic Bubble Tea way so it does not emit terminal control sequences.
MMM MUST NOT implement custom rendering behavior to achieve this.

### File scanning rules

Where commands enumerate the mods folder:
- Only files ending in `.jar` are considered jar candidates.
- Files ending in `.disabled` MUST be ignored.
- `.mmmignore` patterns MUST be applied consistently across `scan`, `install`, and `prune`.

### Missing config

Aside from the `init` command, most commands require a config file to operate.
When the config is missing, MMM MUST use a consistent gate.

Policy:
- When config is missing and MMM is interactive, MMM MUST offer to run `init` and then resume the original command.
- When config is missing and MMM is unattended or non-interactive, MMM MUST fail fast with an actionable error instructing the user to run `mmm init`.

#### Interactive (tty)

```
‼️ No configuration file found at ./modlist.json.
? Initialize now? (y/N):

enter accept • ctrl+c/esc quit
```

On Initialize now:
- Launch the `init` tui-lite flow.
- If init succeeds, resume the original command automatically.
- If init is cancelled or fails, return to the original command and exit safely.

#### Unattended and non-interactive

This applies when `--unattended` is set, or when stdin or stdout is not a TTY.

```
‼️ No configuration file found at ./modlist.json.

Run `mmm init` to create one.
```

The app MUST fail fast with an actionable error.

Message shape:
- "No configuration file found at `<path>`."
- "Run `mmm init` to create one."

### Predictable exit codes

- Exit codes MUST be stable and documented for automation.
- Exit code `0` means success.
- Exit code `1` means the command failed and did not accomplish its intended outcome.
- Exit code `2` means a well-defined no-op or "not applicable" state that is not a failure.
  - Such as `test` when the requested target version equals the configured version.

If a command uses exit code `2`, it MUST be explicitly documented.

### Message quality

- Error messages MUST include:
  - what went wrong
  - the specific input that caused the problem, when applicable
  - the next action the user can take
- MMM MUST NOT require `--debug` for normal comprehension of failures.
- MMM MUST avoid ambiguous phrasing like "failed" without a reason.

### Error handling and recovery

This section defines cross-cutting rules for how MMM communicates failures and recovers safely.
It complements `### Predictable exit codes`, `### Message quality`, and `### No hidden prompts`.

#### Handled errors

Some errors are intentionally handled and printed by MMM so output is not duplicated by Cobra.

Look for:
- `clierrors.IsHandled(err)`
- `cmd.SilenceErrors = true`
- `cmd.SilenceUsage = true`

#### Recovery flows

Some commands MAY recover interactively from expected failures in the interactive terminal (tui-lite).

Example:
- `add` uses a Bubble Tea recovery state machine to resolve unknown platforms, not found mods, and missing compatible files.

#### Unattended and non-interactive contract

In unattended and non-interactive contexts:
- MMM MUST NOT prompt (see `### No hidden prompts`)
- MMM MUST fail fast when required inputs are missing
- Errors MUST be actionable without interactive context

Open questions:
- Which exit codes are part of the stable automation contract beyond `test`
- Which errors should be treated as handled vs unhandled in each command

### Accessibility

- Every interactive path MUST be keyboard-only.
- Color MUST NOT be the only signal. When color is not available, text and symbols MUST still communicate status.
- Animations and constantly updating output SHOULD be minimized and MUST stop with `--quiet`.
- Destructive actions MUST default to safety (see `### Destructive confirmations`).
- Output SHOULD avoid dense decorative output and SHOULD prefer consistent prompt patterns to support screen readers and transcript parsing.

### Mod list visibility

When MMM is working with a list of mods, MMM MUST present the full set of mods at the same time, even if there are hundreds.

- MMM MUST NOT paginate the mod list.
- MMM MUST NOT partially render the mod list.

In tty mode, MMM MAY update per-mod status in place using terminal control sequences, but the view MUST always include all mods.

In non-tty mode, MMM MUST print the mod list as a transcript (no terminal control sequences).

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

### Debug flag

`--debug` is for diagnosing technical issues.
It MUST add additional detail that helps track down the source of a problem.
It MUST NOT be required to understand normal failures.

When `--debug` is set:
- MMM MUST write a debug log file in the same directory as the config file.
- MMM MUST create the log file only when `--debug` is set.
- MMM MUST use a stable, one-event-per-line logfmt format.
- The log MUST provide a full action audit trail for the run.

Logfmt requirements:
- Each line MUST be parseable as logfmt.
- Keys MUST be stable and documented.
- Values MUST avoid control characters.

Minimum fields per line:
- `ts` (RFC3339Nano)
- `level` (debug, info, warn, error)
- `event` (short event name)
- `cmd` (root command name, and subcommand when applicable)

Safety requirements:
- The log MUST NOT include secrets such as api keys, tokens, or auth headers.
- When errors occur, the log SHOULD include enough detail to diagnose without requiring reproduction.

## Language-dependent prompts (option initials)

### Overview

Some prompts ask the user to respond with the initial (or short form) of one of two or more options.
These prompts are language-dependent and MUST NOT be hardcoded to English.

Problem:
- Using `y/n` (or printing `(y/N)`) assumes English.
- In many locales the initials are different, and showing `ja/nein (y/N)` is confusing and unusable.

Scope:
- This applies to prompts where the user is expected to answer using:
  - a short token (typically a single character), or
  - the full option label
- Examples:
  - Yes/No
  - Overwrite/Cancel
  - Retry/Cancel
- This does not apply to Bubble Tea navigation keys (j/k/h/l/q/?/esc/arrows/enter/tab/ctrl+c).
  - Those are treated as universal interaction conventions, not language abbreviations.

#### Contract

##### Option sets

Every language-dependent prompt uses an explicit option set.
Options have stable IDs in code, plus localized display and input tokens from i18n.

Each option must have:
- `label`: what MMM prints to describe the option (and what MMM accepts as full input)
- `short`: the short token MMM prints and accepts (typically one character)

Defaults are behavior and MUST be owned by code, not translations.

##### Rendering

MMM SHOULD render a compact suffix that shows the short tokens for the available options.
The suffix MUST make the default clear.

Recommended style:

```
? <question> (o/c) [default: c]
```

Notes:
- Do not rely on case to communicate defaults. Many scripts do not have case.
- Keep the suffix stable and easy to scan in monospaced terminals.

##### Parsing

Given user input:
- Trim whitespace.
- If the input is empty, select the default option (owned by code).
- Otherwise match case-insensitively against:
  - the option `short`, then
  - the option `label`
- If no option matches:
  - interactive terminal: re-prompt with a localized "invalid choice" message and re-render the options
  - unattended or non-interactive: do not prompt; use the command's defined safe fallback (often cancel/no-op) or require an explicit flag such as `--force`

##### Validation requirements

Within a single option set and locale:
- `short` MUST be non-empty.
- `short` MUST be unique across options.
- `label` MUST be non-empty.

Violations SHOULD fail loudly in developer or CI contexts so mistakes in Crowdin do not ship silently.

##### i18n key guidance

The i18n keys should keep translation authoring simple.
Avoid mini-languages (like `c|cancel`) in translation files.

Suggested naming pattern:
- `<area>.<thing>.question`
- `<area>.<thing>.option.<id>.label`
- `<area>.<thing>.option.<id>.short`

Where `<id>` is the stable option ID in code (example: `overwrite`, `cancel`).

##### Examples

Overwrite / Cancel:

```
? Configuration file already exists at config.json. Overwrite? (o/c) [default: c]
```

Accepted inputs:
- empty input selects default `cancel`
- `o` or `overwrite`
- `c` or `cancel`

Yes / No:

Do not hardcode `(y/N)`.
Define localized `label` and `short` values and render the same suffix style:

```
? Delete these files? (<yesShort>/<noShort>) [default: <noShort>]
```

The parser accepts both the localized short token and the localized full label.

## Confirmations

### Destructive confirmations

When a confirmation prompt is used for a destructive action:
- The default MUST be No (owned by code, not translations).
- Pressing Enter MUST select the default.

The prompt rendering and parsing MUST follow `## Language-dependent prompts (option initials)`.

Guidelines:
- Every confirmation prompt MUST include an explicit default.
- If the default is No, pressing Enter MUST NOT perform the destructive action.
- MMM SHOULD preview what will happen before asking the question.

### Confirmation prompt template

When the app asks a yes or no question:
- The prompt MUST follow `## Language-dependent prompts (option initials)`.
- Destructive actions MUST default to No unless `--force` is present.
- Pressing Enter MUST choose the default.
- EOF MUST abort safely.

## Consistency targets

MMM SHOULD feel consistent across commands:
- Same terms for the same concepts (config, lock file, mods folder, unmanaged)
- Same exit code meanings
- Same prompt style and cancel keys
- Same quiet behavior interpretation

If MMM must break consistency for a specific command, the reason MUST be documented and testable.

## Shared interaction requirements

This section defines shared interaction building blocks used across commands.
Some requirements apply only to the interactive terminal (tui-lite). Others are cross-cutting.

### Tui-lite only

Tui-lite is Bubble Tea based terminal interaction that acts like a traditional terminal app with small UX improvements.
It is NOT a full-screen terminal app.

Rules:
- Answers MUST be echoed on the same line as the prompt.
- The app MUST avoid clearing the screen, hiding previous lines or omitting the line break after user input.
- Transient UI (selection lists) MUST collapse into an answered prompt line after the user confirms.

#### Bubble Tea only

ALL input and output in tui-lite mode MUST use Bubble Tea controls.
The app MUST NOT mix standard input/output with Bubble Tea rendering in tui-lite mode.

#### Footer hints

When the application is waiting for input in interactive mode, it MUST not hide Bubble Tea's footer hint line.
The app reuses Bubble Tea's built-in footer hints without modification.

#### Selection list rendering

Selection lists are allowed as a temporary input affordance.

Visual rules:
- The currently focused option is indicated with a pointer (for example `❯`).
- A selected option is indicated with a colorized `✓` marker.
- Color may be used, but it MUST NOT be the only signal.

Collapse rule:
- After selection is confirmed, the list must collapse to a final answered prompt line that records the choice(s).
- If multiple selections were allowed, the final line MUST list all chosen options in a comma separated format.
- The temporary list UI MUST not be the only record of what was chosen.

Frame snapshot:
```
? Which types of releases would you like to consider to download?
    alpha
❯   beta
  ✓ release

? Which types of releases would you like to consider to download? beta, release
```

### Cross-cutting patterns

#### Progress bars

When MMM renders a progress bar, it MUST be bounded (it MUST show both filled and empty cells).

Default style:
`█████░░░░░`

ASCII fallback:
`[#####-----]`

A progress bar MUST be accompanied by a numeric line showing percent and total, for example:
`50% (512 KB / 1 MB)`

#### Global language

The app MUST use consistent terms:
- config file: `modlist.json`
- lock file: `modlist-lock.json`
- mods folder
- managed file, unmanaged file
- ignore rules: `.mmmignore`

#### Global iconography

The app MUST use consistent icons:
- question prompt: `?` - Using `QuestionStyle`
- Uncertainty: `❔` - Using `QuestionStyle`
- selection indicator: `✓` - Using `SelectedItemStyle`
- focus pointer: `❯` - Using `SelectedItemStyle`
- Mod pinned (skipped): `📌`
- Mod/File preparing status: `⏳`
- Mod/File downloading status: `⬇️`
- Mod/File installed status: `✅`
- Mod/File error status: `❌`
- Command final error summary: `‼️`

##### ASCII fallback

When the terminal does not support Unicode, the app MUST use ASCII fallbacks:
- question prompt: `?`
- Uncertainty: `?`
- selection indicator: `*`
- focus pointer: `>`
- Mod pinned (skipped): `+`
- Mod/File preparing status: `[~]`
- Mod/File downloading status: `->`
- Mod/File installed status: `V`
- Mod/File error status: `X`
- Command final error summary: `!!`

`✅` MUST be used only for final success states.

This applies at the mod or file level, not only at the command level.
If a mod or file is fully installed (and will remain installed), it is a final success for that mod or file even if the overall command later fails.

`❌` MUST be used for mod or file level errors and for listing contributors to a failure.
Final command level error summaries MUST use `‼️` (or `!!` in ASCII mode).

Color rules:
- `‼️` MUST use `ErrorStyle`.
- `!!` MUST also use `ErrorStyle` (red), even in ASCII mode.

For in-progress states, MMM MUST use an indeterminate spinner (animated) instead of printing changing text.

#### Colorization

All output should always be consistently colorized at all times unless `--no-color` is used.

#### Parenthetical metadata styling

When MMM prints secondary metadata in parentheses, the contents of the parentheses MUST use `ParenStyle`.

Example:
`✅ Inventory Sorting (inventory-sorting) is installed`

#### Cancel behavior

Bubble Tea flows:
- `ctrl+c` cancels and exits
- `esc` goes back, or cancels when there is no previous step
- When Bubble Tea provides `q` as a quit key for a given control, the app MUST keep that behavior.
  - The app MUST NOT add or rebind `q` in controls that do not already support it.

Line prompts:
- EOF aborts with no side effects
  - `q` is not a special cancel key in line prompts or free text input.

#### Unmanaged files notice

When a command detects unmanaged jar files and it is not a fatal condition, the app MUST report it consistently.

Frame snapshot:
```
Unmanaged files detected:

There are jar files in your mods folder that are not in your lock file.

❌ unmanaged-A.jar
❌ unmanaged-B.jar

Run `mmm scan` to adopt or resolve these files.
```

#### Cobra and Bubble Tea integration

When there are idiomatic ways to integrate Cobra commands and Bubble Tea flows, the app MUST use them consistently.
Do not reinvent the wheel for behaviour that Cobra and Bubble Tea already support.

## Validation plan

To validate that MMM follows these guidelines:
- Run command flows in non-interactive terminal (non TTY), `--unattended`, interactive tty, and `--quiet` variants.
- Capture outputs and exit codes as evidence.
- Treat divergence between execution contexts as a bug unless explicitly documented.

The current black-box discovery plan and capture harness live under:
- `docs/interactions/black-box-discovery.md`
- `docs/interactions/discovery/README.md`
