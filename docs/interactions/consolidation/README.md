# Consolidation detail

This folder contains the detailed, frame snapshot and message level requirements that back `docs/interactions/interaction-consolidation.md`.

These documents are the implementation handoff for MMM interaction modes:
- Non-interactive terminal
- Interactive terminal (tui-lite)

They also specify how the `--quiet` flag is handled in each mode.

Out of scope:
- Full root-level TUI launched from `mmm` with no args

## Tui-lite definition

Tui-lite is transcript-first.
It MUST feel like traditional terminal input and output, not a full-screen app.

Tui-lite is the interactive terminal mode.
It is not a separate layer.
It is traditional terminal input and output with a small amount of Bubble Tea assistance for better selection and recovery UI.

Key rule:
- Prompts appear on lines.
- Answers are echoed on the same line.

Selection list exception:
- A temporary list UI is allowed for picking values.
- Once confirmed, it MUST collapse to an answered prompt line.

## Fidelity requirement

For interactive and non-interactive terminal modes, and for `--quiet` behavior, this folder MUST specify what the user sees as terminal frame snapshots.
Each frame snapshot includes the transcript so far, the active control, and the footer hints.
The command used segment records how MMM was invoked.

## Contents

- `docs/interactions/consolidation/shared.md`
- `docs/interactions/consolidation/root.md`
- `docs/interactions/consolidation/help.md`
- `docs/interactions/consolidation/init.md`
- `docs/interactions/consolidation/add.md`
- `docs/interactions/consolidation/install.md`
- `docs/interactions/consolidation/update.md`
- `docs/interactions/consolidation/list.md`
- `docs/interactions/consolidation/scan.md`
- `docs/interactions/consolidation/remove.md`
- `docs/interactions/consolidation/test.md`
- `docs/interactions/consolidation/change.md`
- `docs/interactions/consolidation/prune.md`
