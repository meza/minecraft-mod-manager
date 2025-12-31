# Shared interaction requirements

This document defines shared interaction building blocks used across commands.

These requirements apply to tui-lite only.

## Tui-lite transcript rules

Tui-lite is not a full-screen app.
It is traditional terminal input and output with small UX improvements.

The transcript MUST be the primary artifact.
If a user copies and pastes the run output into an issue, it must remain readable and complete.

Rules:
- Prompts MUST be plain console lines.
- Answers MUST be echoed on the same line as the prompt.
- MMM MUST avoid clearing the screen or hiding previous lines.
- Transient UI (selection lists) MUST collapse into an answered prompt line after the user confirms.

## Frame snapshot format (hard requirement)

Every interactive and non-interactive state MUST be specified as a terminal frame snapshot.
This is not optional.
If a state is described only in prose, the implementation is under-specified.

A frame snapshot is a copy/pasteable block that shows exactly what the user sees at that moment.

Frame snapshot requirements:
- Frame snapshots MUST include only MMM output and UI.
  - Do not include the shell prompt or the command invocation line inside the frame snapshot.
- Include the transcript so far.
- If the state uses a selection list, include the temporary list UI exactly as shown.
- Include the footer hint line when the state expects input.
- Use blank lines intentionally. If spacing matters, it MUST be shown.

Frame naming:
- Each frame MUST be labeled with a stable state id, for example `INIT-03`.
- Each branch MUST be transcribed as a sequence of frames.

Non-interactive frame requirements:
- Include `--non-interactive` in the command used segment.
- There is no footer hint line, because MMM is not waiting for input.

Quiet frame requirements:
- Include `--quiet` in the command used segment.
- If output is intentionally silent, the frame MUST still record that fact:
  - `Output: none`
  - `Exit code: <n>`

## Command used segments (hard requirement)

Every run sequence MUST include a `Command used` segment in Markdown.
This is the place to record how MMM was invoked without implying MMM printed it.

The command used value is the arguments you would pass after `mmm` or `mmm.exe`.
For running MMM with no arguments, use `<no args>`.

Format:
```
#### Command used
`<flags> <command> <args>`
```

## State coverage requirement

Every command consolidation doc MUST:
- Define its state model, including cancellation and key error states.
- Provide frame snapshots that cover every defined state at least once.
- Provide complete run sequences, from the command used segment to the terminal end state, for each documented branch.

This is a handoff requirement.
If any state is not transcribed, the implementation is under-specified.

## Transcript conventions

Placeholders:
- Use `<like this>` for values that depend on environment or user data.

User input:
- User input appears after the prompt on the same line.

Frame snapshot:
```
? Initialize now? (y/N): y
```

Cancellation:
- `ctrl+c` cancels.
- MMM MUST print `Cancelled.` and exit with code 1 unless a command defines a different cancellation contract.

Frame snapshot:
```
^C
Cancelled.
```

Silent outcomes:
- Some `--quiet` success paths are intentionally silent.
- When documenting a silent outcome, represent it as:
  - "Output: none"
  - "Exit code: <n>"

Prompt prefix:
- Use `?` as the prompt prefix.

Frame snapshot:
```
? Which loader would you like to use? fabric
```

## Footer hints

When MMM is waiting for input in interactive mode, it MUST not hide Bubble Tea's footer hint line.
MMM reuses Bubble Tea's built-in footer hints without modification.

## Selection list rendering

Selection lists are allowed as a temporary input affordance.
They are the only place where tui-lite is allowed to render more than one line of UI for a single question.

Visual rules:
- The currently focused option is indicated with a pointer (for example `❯`).
- A selected option is indicated with a marker (for example `✓`).
- Color may be used, but it MUST NOT be the only signal.

Collapse rule:
- After selection is confirmed, MMM MUST print a final answered prompt line that records the choice.
- The temporary list UI MUST not be the only record of what was chosen.

Frame snapshot:
```
? Which types of releases would you like to consider to download?
    alpha
❯   beta
  ✓ release

? Which types of releases would you like to consider to download? beta,release
```

## Global language

MMM MUST use consistent terms:
- config file: `modlist.json`
- lock file: `modlist-lock.json`
- mods folder
- managed file, unmanaged file
- ignore rules: `.mmmignore`

## Missing config gate

When a command requires config and config is missing, MMM MUST use a consistent gate.

### Interactive (tty)

Frame snapshot:
```
No configuration found.
MMM looked for a config file at: <path>
This command needs config to continue.
? Initialize now? (y/N):

enter accept • ctrl+c/esc quit
```

If the user answers No:
```
? Initialize now? (y/N): n
```
MMM exits with no side effects.

On Initialize now:
- Launch the `init` tui-lite flow.
- If init succeeds, resume the original command automatically.
- If init is cancelled or fails, return to the original command and exit safely.

### Non-interactive

MMM MUST fail fast with an actionable error.

Message shape:
- "No configuration file found at `<path>`."
- "Run `mmm init` to create one."

## Quiet flag decision rule

With `--quiet` set, MMM prints only actionable results.

Actionable means one of:
- the command failed
- the user must take a follow up action to reach their goal
- the command exists to display information as the primary output (list is the exception)

## Confirmation prompt template

When MMM asks a yes or no question:
- The prompt MUST show the default explicitly, for example `(y/N)` or `(Y/n)`.
- Destructive actions MUST default to No unless `--force` is present.
- Pressing Enter MUST choose the default.
- EOF MUST abort safely.

Template:
```
? <question> (y/N):
```

## Cancel behavior

Bubble Tea flows:
- `ctrl+c` cancels and exits
- `esc` goes back, or cancels when there is no previous step
- When Bubble Tea provides `q` as a quit key for a given control, MMM MUST keep that behavior.
  - MMM MUST NOT add or rebind `q` in controls that do not already support it.

Line prompts:
- EOF aborts with no side effects
  - `q` is not a special cancel key in line prompts or free text input.

## Unmanaged files notice

When a command detects unmanaged jar files and it is not a fatal condition, MMM MUST report it consistently.

Frame snapshot:
```
Unmanaged files detected:
MMM found jar files in your mods folder that are not in your lock file.
- mods/unmanaged.jar
Run `mmm scan` to adopt or resolve these files.
```

Frame snapshot:
```
Unmanaged files detected:
- mods/unmanaged.jar
Run `mmm scan` to adopt or resolve these files.
```
