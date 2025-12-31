# mmm-151 interaction audit

This document records the current observed interaction behavior across MMM commands using a pseudo TTY harness.
It is not spec authority.
Use `docs/interactions/interaction-guidelines.md` and `docs/interactions/interaction-consolidation.md` as the target interaction contract.
Legacy specs in `docs/specs/` may not match the desired future behavior.

## Harness

Runs used:
- MMM binary: `/work/build/linux/amd64/mmm`
- Workspace: `/tmp/mmm-interactions`
- In repo snapshot of outputs: `docs/interactions/audit-artifacts/mmm-151/README.md`
- TTY capture: expect scripts under `docs/interactions/discovery/expect/`
- Telemetry disabled for runs: `MMM_DISABLE_TELEMETRY=1`

Modes covered per scenario:
- Non interactive: `--non-interactive`
- Pseudo TTY: expect spawned pty with terminal probe replies
- Quiet variants: `--quiet`

## Scenarios executed

The scenario definitions live in `docs/interactions/black-box-discovery.md`.

The executed run folders are under:
- `/tmp/mmm-interactions/s1-empty`
- `/tmp/mmm-interactions/s2-lifecycle-v2`
- `/tmp/mmm-interactions/s4-prune-v2`
- `/tmp/mmm-interactions/s5-add-recovery`
- `/tmp/mmm-interactions/s6-init-exists`
- `/tmp/mmm-interactions/s7-unmanaged-block`
- `/tmp/mmm-interactions/s8-ignore`
- `/tmp/mmm-interactions/s9-change-noop`

Curated copies of these outputs are stored under:
- `docs/interactions/audit-artifacts/mmm-151/runs/`

## Findings

### Mod IDs and repeatability

- Modrinth slug correction: SoundsBeGone is reachable as `soundsbegone`, not `soundbegone` (404 on Modrinth API).
- MMM accepts Modrinth slugs and stores the slug in config and lock, even when the download URL uses the project id form.

### Terminal mode gating and prompting

- `scan` with missing config in a TTY prompts to init:
  - "No configuration file found at ./modlist.json."
  - "Initialize a new configuration file now? (y/N):"
- `add` uses recovery TUI in a TTY for expected errors.
  Example for unknown platform:
  - "? The platform you entered (not-a-platform) is not a valid platform."
  - Offers retry with `curseforge` or `modrinth` or cancel.
- In `--non-interactive`, the same errors are fail fast and actionable.
  Example:
  - "Unknown platform \"not-a-platform\". Please use one of the following: curseforge, modrinth"

### Confirmation defaults and visibility

- `prune` confirmation prompt does not display an explicit default.
  Observed prompt text:
  - "? Do you want to delete these files?"
  This conflicts with the interaction pattern guidance that defaults must be explicit in the prompt string.

### Exit codes and behavior drift

- `change` no-op behavior differs by terminal mode.
  - Non interactive: `mmm --non-interactive change 1.21.6` exits with code 2 and prints "The target version 1.21.6 is the same as your current version."
  - Pseudo TTY: `mmm change 1.21.6` proceeded with a reinstall path and exited 0 in the audited run.
  This indicates either a spec drift in interactive contexts or a bug in the same-version check when prompts and TUI are enabled.

### Unmanaged files and install behavior

- `scan` reports unknown unmanaged jars in config present flows.
  Example:
  - "Unknown files:"
  - "Could not match unmanaged.jar"
- `install` did not block on an unknown unmanaged jar in the audited run.
  With `mods/unmanaged.jar` present, `mmm --non-interactive install` still reported success.
  This conflicts with `docs/specs/install.md` which describes unmanaged files halting the process until resolved.

### Ignore rules

- `.mmmignore` is respected by `scan`, `install`, and `prune` in the audited run.
  With `mods/ignored-by-pattern.jar` and `.mmmignore` containing that filename:
  - `scan` reported all mods managed
  - `prune` reported "No unmanaged files found."
- `*.jar.disabled` is implicitly ignored by these commands because file enumeration is limited to `*.jar`.

## Follow up candidates

These are candidate follow ups under `mmm-151`.

- Make destructive prompts show explicit defaults, especially `prune` confirmation.
- Decide whether `change` should be a strict no-op for same-version targets in all terminal modes and make behavior consistent.
- Resolve `install` handling of unknown unmanaged jar files to match the desired automation and safety contract.
- Decide whether scan without config should support a non-interactive path, or should always require an init step.
