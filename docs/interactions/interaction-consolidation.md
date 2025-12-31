# Interaction consolidation requirements

This document is a detailed, normative interaction requirement specification for MMM.
It defines how each command flow MUST behave so the product aligns with `docs/interactions/interaction-guidelines.md`.

Frame snapshot and message level requirements live under:
- `docs/interactions/consolidation/README.md`

Implementation reading order (to avoid guessing):
1. `docs/interactions/interaction-guidelines.md` (ethos and global rules)
2. `docs/interactions/interaction-consolidation.md` (this file, per-command contract)
3. `docs/interactions/consolidation/README.md` and `docs/interactions/consolidation/shared.md` (frame snapshots, footers, cancel keys, shared building blocks)
4. `docs/interactions/consolidation/*.md` (per-command, state-by-state transcripts)
5. `docs/interactions/patterns/*` and `docs/interactions/execution-tiers.md` (cross-cutting patterns and mode gating)

## Rendering model

MMM interactions are transcript-first across interactive and non-interactive modes.
They MUST feel like traditional terminal input and output, with small enhancements for interactive selection and recovery.

Key requirements:
- Prompts MUST appear as normal console lines.
- Answers MUST be echoed on the same line as the prompt.
- The interaction transcript MUST remain readable when copied into a bug report.

Selection list exception:
- When a selection list is used (for example, multi-select release types), the selection UI may be rendered temporarily.
- Once the user confirms the selection, the UI MUST collapse into an answered prompt line so the transcript reads as a sequence of questions and answers.

Frame snapshot requirement:
- All interactive and non-interactive outputs MUST be specified as frame snapshots under `docs/interactions/consolidation/`.

Scope:
- Non-interactive terminal execution
- Interactive terminal execution (tui-lite only)
- `--quiet` flag behavior across both

Out of scope:
- The future full root-level TUI launched from `mmm` with no arguments
- Visual design and branding

## Authority model

This document defines the target interaction contract for MMM.
Legacy specs and user docs may disagree with this contract.

When there is disagreement:
- MMM MUST converge to this contract for interaction behavior.
- Any intentional mismatch MUST be explicitly recorded as a known gap and validated.

## Terms

- Config: `modlist.json`
- Lock: `modlist-lock.json`
- Mods folder: the directory configured in the config where jar files live
- Managed file: a jar file that is present in the lock file as an installation record
- Unmanaged file: a jar file in the mods folder that is not present in the lock file
- Ignore rules: patterns from `.mmmignore` applied relative to the mods folder
- TTY: a terminal device. If stdout is redirected, it is not a TTY.
- Interactive (tui-lite): per-command Bubble Tea flows and wrappers plus simple line prompts

Normative keywords:
- MUST, MUST NOT, SHOULD, SHOULD NOT, MAY

## Global interaction contract

### Terminal mode gating

1. If `--non-interactive` is set, MMM MUST run in non-interactive mode.
2. If prompts would be required but stdin or stdout is not a TTY, MMM MUST NOT prompt.
3. Non-TTY usage is treated as non-interactive by default.

### Quiet flag

`--quiet`:
- MUST suppress non-essential output
- MUST still print errors
- MUST print only actionable results

`--debug`:
- MAY emit additional diagnostics
- MUST NOT be required to understand normal failures

### Output streams

- Required results SHOULD go to stdout.
- Errors and warnings SHOULD go to stderr.
- Interactive rendering MUST NOT produce terminal control sequences when stdout is not a TTY.

### Exit code policy

- `0`: success
- `1`: failure
- `2`: defined no-op or not applicable

If a command uses exit code `2`, it MUST be documented in that command section.

### Cancellation

Interactive flows MUST support explicit cancel.

- Bubble Tea flows:
  - `ctrl+c` MUST cancel and exit
  - `esc` MUST go back or cancel when there is no previous step
  - When Bubble Tea provides `q` as a quit key for a given control, MMM MUST keep that behavior.
- Line prompts:
  - EOF MUST abort the operation safely
  - If the prompt is for a destructive action, abort MUST behave as "do not proceed"

### Confirmation prompts

See `docs/interactions/patterns/confirmations.md`.

Minimum requirements:
- Confirmation prompts MUST include an explicit default like `y/N` or `Y/n`.
- Destructive actions MUST default to No unless `--force` is used.
- MMM SHOULD show a preview of what will be deleted or changed.

### File scanning rules

Where commands enumerate the mods folder:
- Only files ending in `.jar` are considered jar candidates.
- Files ending in `.disabled` MUST be ignored.
- `.mmmignore` patterns MUST be applied consistently across `scan`, `install`, and `prune`.

## Flow requirements by command

Each command section defines:
- User goal
- Required behavior
- Behavior by terminal mode
- Quiet flag requirements
- Exit codes

### `help`

User goal:
- Understand how to use MMM and its commands

Detail:
- `docs/interactions/consolidation/help.md`

Required behavior:
- `mmm --help` and `mmm help` MUST print the root help content and exit 0.
- `mmm <command> --help` MUST print command help and exit 0.
- Help output MUST be readable without color.

Terminal mode behavior:
- Non-interactive terminal: print help and exit 0.
- Interactive terminal (tui-lite): print help and exit 0. No interactive UI.

Quiet flag:
- `--quiet` MUST NOT suppress help output. Help is a required result.

### Root invocation with no args

User goal:
- Discover available commands quickly

Required behavior:
- Running `mmm` with no arguments MUST print help and exit 0 until the full root TUI exists.
- It MUST NOT launch a full-screen UI in this scope.

Detail:
- `docs/interactions/consolidation/root.md`

Quiet flag:
- Help is a required result and MUST still print.

### `init`

User goal:
- Create a valid config and lock file for a new working directory

Required behavior:
- MMM MUST determine the config path (default `./modlist.json` or `--config`).
- MMM MUST write a config and create an empty lock file.
- MMM MUST validate:
  - the mods folder exists and is a directory
  - the Minecraft version is valid
- If the game version is omitted and defaults to `latest`:
  - MMM SHOULD fetch the latest stable Minecraft release from the official API
  - If the API is unavailable and MMM is non-interactive, MMM MUST fail with an actionable error instructing the user to pass `--game-version`

Interactive behavior:
- If the config file exists and MMM is interactive, MMM MUST ask whether to overwrite or allow choosing a new file name.
- MMM MUST gather required values through flags or interactive selection:
  - loader
  - Minecraft version
  - allowed release types
  - mods folder
- Invalid values MUST be recoverable in the interactive flow.

Non-interactive behavior:
- If the config file exists, MMM MUST exit with an error.
- If required values are missing and cannot be defaulted safely, MMM MUST exit with an error.

Quiet flag:
- With `--quiet`, MMM MAY print nothing on success.
- Errors MUST still print.

Exit codes:
- 0 on success
- 1 on failure

Detail:
- `docs/interactions/consolidation/init.md`

### `add`

User goal:
- Add a mod to the config and download its jar to the mods folder

Required behavior:
1. MMM MUST ensure config and lock exist.
2. MMM MUST resolve the requested mod `<platform> <id>` for the configured loader and game version.
3. MMM MUST download the selected file into the mods folder.
4. MMM MUST append:
   - a mod entry to the config
   - an installation record to the lock

Error handling requirements:
- Unknown platform MUST be recoverable in interactive mode and fail fast in non-interactive mode.
- Missing project or no compatible file MUST offer a recovery path in interactive mode:
  - try the alternate platform
  - enter a different id
  - cancel
- Download failures MUST fail the command with an error.

Terminal mode behavior:
- Non-interactive terminal: MUST not prompt. MUST return an actionable error on unknown platform, missing id, or incompatibility.
- Interactive terminal (tui-lite): MAY use a Bubble Tea recovery flow for expected failures. It MUST still be cancellable.

Quiet flag:
- On success, MMM MAY emit no output.
- On failure, MMM MUST emit an error.

Exit codes:
- 0 on success
- 1 on failure

Detail:
- `docs/interactions/consolidation/add.md`

### `list`

User goal:
- See what is configured and whether it is installed correctly

Required behavior:
- MMM MUST load config and lock.
- MMM MUST sort entries alphabetically.
- For each mod, MMM MUST indicate installed vs missing or mismatched:
  - Installed means lock entry hash matches a local file.
  - Missing means missing file, missing lock entry, or hash mismatch.
- When output is not colorized, MMM MUST use plain text markers such as `V` and `X`.
- When hash mismatch occurs, MMM SHOULD include a short hint to run `mmm install`.

Terminal mode behavior:
- Non-interactive terminal: print the list and exit. No prompts.
- Interactive terminal (tui-lite): MAY use a tui-lite wrapper for rendering, but MUST not change semantics.
  - The user MUST be able to exit without side effects.

Quiet flag:
- The list output is a required result and MUST still print under `--quiet`.

Exit codes:
- 0 on success
- 1 on failure (missing or invalid config)

Detail:
- `docs/interactions/consolidation/list.md`

### `install`

User goal:
- Ensure every configured mod is downloaded and matches the lock file

Required behavior:
1. MMM MUST load config and lock.
2. MMM MUST scan the mods folder for unmanaged files.
   - If unmanaged files exist, MMM MUST proceed with install and then report unmanaged files as actionable information.
   - The remediation MUST be "Run `mmm scan` to adopt or resolve these files."
3. For each configured mod:
   - If a lock entry exists, MMM MUST verify hash and redownload if missing or mismatched.
   - If no lock entry exists, MMM MUST fetch metadata and download a compatible file, then write a lock entry.
4. MMM MUST update the lock file and persist any name changes in the config.

Terminal mode behavior:
- Non-interactive terminal: no prompts. All outcomes communicated via logs and exit codes.
- Interactive terminal (tui-lite): MAY use a tui-lite log wrapper to show progress, but MUST not require user input.

Quiet flag:
- MMM SHOULD suppress progress logs.
- MMM SHOULD be silent on success when there are no unmanaged files detected.
- If unmanaged files exist, MMM MUST print the unmanaged file list even with `--quiet` set.

Exit codes:
- 0 on success
- 1 on failure

Detail:
- `docs/interactions/consolidation/install.md`

### `update`

User goal:
- Upgrade configured mods to newer compatible releases

Required behavior:
1. MMM MUST run `install` first.
   - If unmanaged files exist, the install phase MUST still proceed and the unmanaged list MUST be reported.
2. For each configured mod that is not pinned to a specific version:
   - MMM MUST query for a newer compatible file.
   - If found, MMM MUST download the new jar, remove the previous jar, and update the lock entry.
3. MMM MUST keep mod names in the config in sync with observed metadata.
4. On download failure for a mod, MMM MUST keep the previous version on disk and MUST NOT alter the lock entry for that mod.

Terminal mode behavior:
- Non-interactive terminal: no prompts.
- Interactive terminal (tui-lite): MAY use a tui-lite log wrapper.

Quiet flag:
- MMM SHOULD suppress per-mod progress.
- MMM SHOULD be silent on success when there are no unmanaged files detected.
- If unmanaged files exist, MMM MUST print the unmanaged file list even with `--quiet` set.
- Errors MUST still print.

Exit codes:
- 0 on success
- 1 on failure

Detail:
- `docs/interactions/consolidation/update.md`

### `remove`

User goal:
- Remove mods from config and lock, and delete their jars when present

Required behavior:
1. MMM MUST resolve provided names or ids against the config, supporting glob patterns.
2. If an installation exists, MMM MUST delete the jar and remove the lock entry.
3. MMM MUST remove the mod entry from the config.
4. With `--dry-run`, MMM MUST report intended actions without writing changes or deleting files.

Terminal mode behavior:
- Non-interactive terminal: no prompts.
- Interactive terminal (tui-lite): no prompts.

Quiet flag:
- On success, MMM MAY emit no output.
- Errors MUST still print.

Exit codes:
- 0 on success
- 1 on failure

Detail:
- `docs/interactions/consolidation/remove.md`

### `scan`

User goal:
- Identify unmanaged jar files and optionally adopt them into MMM management

Required behavior:
1. MMM MUST enumerate jar candidates in the mods folder:
   - ignore `.disabled`
   - respect `.mmmignore`
   - skip files already managed by the lock
2. MMM MUST attempt to identify each file by hash:
   - try preferred platform first (default `modrinth`)
   - fall back to the alternate platform when there are no hits
3. MMM MUST present results grouped into:
   - recognized
   - unknown
   - unsure (platform lookup failed)
4. Write behavior:
   - With `--add`, MMM MUST update config and lock to include discovered mods only if there are no unsure results.
   - Without `--add`, MMM MUST prompt in interactive mode before writing changes.
   - In non-interactive mode without `--add`, MMM MUST not write changes.

Missing config behavior:
- If no config exists:
  - Interactive: MMM MUST tell the user and ask whether to initialize config now.
    - If user confirms, MMM MUST run the init tui-lite flow and then continue scanning.
  - Non-interactive: MMM MUST fail with an error that instructs the user to run `mmm init` first.

Terminal mode behavior:
- Non-interactive terminal: no prompts. Write only when `--add` is set and safe to do so.
- Interactive terminal (tui-lite): MAY use line prompts for confirmation. It MUST be safe to cancel.

Quiet flag:
- Non-essential progress logs MUST be suppressed.
- With `--quiet` set, MMM MUST print only actionable results:
  - Unknown files
  - Unsure files
  - Any reason the scan could not be applied

Exit codes:
- 0 on success
- 1 on failure

Detail:
- `docs/interactions/consolidation/scan.md`

### `test`

User goal:
- Determine whether all configured mods support a target Minecraft version

Required behavior:
1. MMM MUST resolve the target version from the argument or `latest`.
2. MMM MUST query each configured mod for compatibility with the target version and configured loader.
3. MMM MUST make no changes to config, lock, or mods folder.
4. MMM MUST print:
   - success when all mods support the target version
   - missing mods list when any mod lacks support

Terminal mode behavior:
- Non-interactive terminal: no prompts.
- Interactive terminal (tui-lite): no prompts. A tui-lite wrapper MAY be used for rendering but MUST not change semantics.

Quiet flag:
- With `--quiet` set, MMM MUST be silent on success.
- On failure, MMM MUST print only the missing mods list.

Exit codes:
- 0 when all mods support the target
- 1 when one or more mods lack support
- 2 when the target version equals the configured version

Detail:
- `docs/interactions/consolidation/test.md`

### `change`

User goal:
- Switch the configured Minecraft version and reinstall mods for that version

Required behavior:
1. MMM MUST treat same-version targets as a no-op.
   - It MUST exit with code 2 and make no changes in every terminal mode.
2. MMM MUST verify mod support for the target version using the same checks as `test`, unless `--force` is used.
3. MMM MUST remove currently installed mod jars and update config `gameVersion`.
4. MMM MUST run `install` to download compatible versions for the new game version.

`--force` behavior:
- MMM MUST proceed even if some mods do not support the target.
- Unsupported mods MUST be skipped during install.
- MMM MUST still change the config version.

Terminal mode behavior:
- Non-interactive terminal: no prompts.
- Interactive terminal (tui-lite): MAY use a tui-lite log wrapper.

Quiet flag:
- MMM SHOULD suppress progress logs.
- Errors MUST still print.

Exit codes:
- 0 on success
- 1 on failure
- 2 when the target version equals the configured version

Detail:
- `docs/interactions/consolidation/change.md`

### `prune`

User goal:
- Delete unmanaged jar files from the mods folder

Required behavior:
1. MMM MUST load config and lock to determine managed files.
2. MMM MUST scan for unmanaged jar files, applying `.mmmignore` and ignoring `.disabled`.
3. If no unmanaged files exist, MMM MUST print a short message and exit successfully.
4. Deletion behavior:
   - With `--force`, MMM MUST delete unmanaged files without prompting.
   - Without `--force`, MMM MUST require confirmation in interactive mode.

Non-interactive safety behavior:
- In `--non-interactive` mode without `--force`, MMM MUST:
  - print an informational message
  - list unmanaged files
  - skip deletion
  - exit successfully

No TTY safety behavior:
- If prompts would be required but stdin or stdout is not a TTY and `--force` is not set, MMM MUST:
  - print an informational message
  - exit without deleting

Terminal mode behavior:
- Non-interactive terminal: no prompts, no deletion unless `--force`.
- Interactive terminal (tui-lite): uses a line confirmation prompt.

Quiet flag:
- `--quiet --force` MUST delete unmanaged files with no output on success.
- Errors MUST still print.

Exit codes:
- 0 on success, including "nothing to prune" and non-interactive skip without deletion
- 1 on failure

Detail:
- `docs/interactions/consolidation/prune.md`

## Cross-flow consistency requirements

### Empty folder behavior

In a folder without config:
- In interactive mode, every command MUST:
  - explain that config is missing and where MMM looked
  - offer to run `mmm init` now
  - if the user accepts, run init and then resume the original command
  - if the user declines, exit without side effects
- In non-interactive mode, every command MUST fail fast with an actionable error instructing the user to run `mmm init`.
- `add` MUST follow the same rule. In interactive mode it offers init and then continues. In non-interactive mode it fails.

### Same concept, same term

MMM MUST use consistent terms in user-facing text:
- config file, lock file, mods folder
- managed and unmanaged
- ignore rules via `.mmmignore`

### Interaction parity across terminal modes

If a command outcome is a no-op or safety refusal, it MUST behave the same way in interactive and non-interactive terminal modes.
Interactive rendering MUST never change the underlying decision.

## Validation requirements

To validate compliance with this document:
- Run each command flow in:
  - interactive tty mode
  - `--non-interactive` mode
  - `--quiet` variants for both
- Capture stdout, stderr, and exit codes.
- Treat any mismatch between required behavior and observed behavior as a bug.

The black-box plan and harness are the canonical validation tools:
- `docs/interactions/black-box-discovery.md`
- `docs/interactions/discovery/README.md`
