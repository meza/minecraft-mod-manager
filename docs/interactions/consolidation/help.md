# Command help frame snapshots

User goal:
- Understand how to use a specific command quickly and correctly.

All requirements in `docs/interactions/consolidation/shared.md` apply.

## State model

States:
- HELP-01: print command help
- HELP-02: exit 0

## Requirements

- `mmm <command> --help` MUST print help for that command and exit 0.
- Help MUST be readable without color.

Help content MUST include:
- One short description of what the command does.
- One minimal usage snippet.
- Required arguments, if any.
- Command-specific flags, if any.
- Interaction notes:
  - what happens in interactive mode
  - what changes under `--non-interactive`
  - what quiet means for this command

## Frame snapshots

### HELP-01 Help output

#### Command used
`scan --help`

```
mmm scan

Find jar files in your mods folder that MMM does not manage.

Usage:
  mmm scan [flags]

Notes:
  In interactive mode, MMM may ask before writing changes.
  With --non-interactive, MMM will not prompt.
  With --quiet, MMM prints only actionable results.
```

## Quiet flag

Help MUST still print.

#### Command used
`--quiet scan --help`

```
mmm scan
...
```

## Non-interactive mode

Non-interactive mode MUST NOT prompt. Help MUST still print.

#### Command used
`--non-interactive scan --help`

```
mmm scan
...
```
