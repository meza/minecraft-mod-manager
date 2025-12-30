# Execution tiers

This document describes how MMM decides which interaction layer to use.
It describes current behavior and notes future requirements when relevant.

## Tier 1: Non-interactive

When it applies:
- `--non-interactive` is set
- Or stdin or stdout is not a TTY in contexts where prompts would otherwise be used

What it means:
- The command must not prompt for missing input
- Missing required inputs must fail fast

Primary user intent:
- Scripting and automation

## Tier 2: Prompted CLI

When it applies:
- Prompts are enabled
- stdin and stdout are TTYs
- A command chooses to ask for a simple confirmation or a missing value without launching Bubble Tea

What it means:
- The command can ask for missing input using line prompts
- The prompts must have clear defaults and clear cancel behavior

Where it exists today:
- `scan` confirmation prompts
- `init` overwrite behavior when config already exists

## Tier 3: Per-command Bubble Tea TUI

When it applies:
- Prompts are enabled
- stdin and stdout are TTYs
- The command implements a Bubble Tea flow or wrapper

What it means:
- The command owns a state machine with explicit cancel and back behavior
- Keyboard-only navigation must be supported

Where it exists today:
- `add` recovery flow
- `init` wizard
- `install`, `list`, `test`, `update` use Bubble Tea wrappers for interactive output

## Future tier: Full root-level TUI

The design doc describes a full TUI that launches when you run `mmm` with no arguments.
That is not implemented yet.

Track this as a future requirement:
- Target model: `docs/tui-design-doc.md`
- Current behavior: prints help on `mmm` with no args

