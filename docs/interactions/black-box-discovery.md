# Black box discovery plan

This document defines how we will validate MMM interactions by running the shipped binary and recording observable behavior.
It is design documentation, not spec authority.
Behavior authority remains in `docs/specs/` and user docs in `docs/commands/`.

## Goals

- Discover and document current interaction behavior across commands and tiers.
- Produce a repeatable run matrix that captures output, exit codes, prompts, and side effects.
- Create a shared baseline for `mmm-151` follow up work.

## Non goals

- Rewriting or overriding behavior specs.
- Implementing fixes, refactors, or new interaction surfaces in this step.

## Approach

- Use the MMM binary to perform all mod resolution and downloads.
- Do not call Modrinth or CurseForge APIs directly.
- Run each scenario in an isolated folder under `/tmp/mmm-interactions` so configs never leak across runs.
- Capture both TTY and non TTY behavior:
  - Non TTY validates the automation contract.
  - Pseudo TTY validates TUI and prompted behavior using `expect` for discovery and capture.

## Test matrix

Each scenario runs in these modes.

- Mode A: pseudo TTY with `expect` and prompts enabled
- Mode B: non interactive CLI with `--non-interactive`
- For both modes: rerun with `--quiet`

Optional drift check:
- Mode C: non TTY without `--non-interactive` by redirecting stdout to confirm prompt suppression and fail fast behavior

## Folder layout

Each run uses a fresh folder.

- Root: `/tmp/mmm-interactions/<scenario>/<mode>/<variant>/`
- Each run owns:
  - `./modlist.json`
  - `./modlist-lock.json`
  - `./mods/`

## Mod set

Use these mods for all scenarios.

- SoundsBeGone: Modrinth `soundbegone`, CurseForge `874633`
- Inventory Sorting: Modrinth `inventory-sorting`, CurseForge `325471`
- Disable Christmas Chests: Modrinth `bhuHzAIs`, CurseForge `945775`
- Restart Detector: Modrinth `restart-detector`

## Scenarios

### Scenario 1: Empty folder commands

In a fresh folder with no config and no lock:

- Run `mmm install`
- Run `mmm update`
- Run `mmm list`

Capture:
- Exit code per command
- Stderr and stdout output
- Whether any files are created

### Scenario 2: Full lifecycle with version changes

Setup:
- `mmm init` for fabric with gameVersion `1.21.6` and modsFolder `./mods`
- Add the mod set using `mmm add` with an explicit platform choice
- Variant A: add via Modrinth first for mods that exist there
- Variant B: add via CurseForge first for mods that exist there

Flow:
- `mmm list`
- Remove a mod, then add it back from the alternate source:
  - Target mod: SoundsBeGone
- `mmm test 1.21.10`
- `mmm change 1.21.10`
- `mmm change 1.21.11`

Capture:
- Exit codes, especially for `test` and `change`
- Whether any step prompts
- Whether any step uses a TUI log wrapper
- What changes in `modlist.json`, `modlist-lock.json`, and `./mods/`

### Scenario 3: Scan decline and unknown jar

Setup:
- `mmm init` for fabric with `1.21.6`
- Add the mod set

Flow:
- Delete `modlist.json` and `modlist-lock.json`
- Run `mmm scan` and decline any init or persist action
- Create a random empty jar in the mods folder:
  - `./mods/random-empty.jar` with 0 bytes
- Run `mmm scan` again

Capture:
- Whether scan offers to init when config is missing
- Whether declining leaves the folder unchanged
- How unknown jars are grouped and reported

### Scenario 4: Unmanaged jar and prune

Setup:
- `mmm init` for fabric with `1.21.6`
- Add the mod set

Flow:
- Copy one installed jar out of `./mods/` to a backup location
- `mmm remove` the same mod
- Copy the backup jar back into `./mods/` so it is unmanaged
  - Target mod: SoundsBeGone
- Run `mmm scan` to confirm the file is unmanaged
- Run `mmm prune` and observe the confirmation prompt
- Rerun `mmm prune` and confirm deletion

Capture:
- Scan output for unmanaged files
- Prune defaults and cancel behavior
- Difference between `--non-interactive`, prompt disabled, and `--force`

## Captures and artifacts

For every run, record:

- Working directory path
- Full command line, including global flags
- Environment notes: at minimum `TERM` and whether output is a TTY
- Exit code
- Raw stdout and stderr
- A readable view of captured TTY output if a TUI is involved
- Final file inventory of the run folder

## Reporting format

For each command in each scenario and mode, summarize:

- Tier selected: non interactive, prompted CLI, or Bubble Tea
- Prompt or TUI surfaced: yes or no
- Default on destructive confirmation: safe or unsafe
- Cancel and quit semantics observed
- User visible recovery guidance quality
- Side effects on disk

