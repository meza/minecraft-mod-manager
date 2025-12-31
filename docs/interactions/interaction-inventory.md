# Interaction inventory

This document inventories the current interaction surfaces and where they live.
It is a map, not a spec.

## Primary user surfaces

### CLI commands

Root command and global flags:
- Code: `cmd/mmm/root.go`
- Entrypoint: `main.go`
- User guide: `README.md`

Subcommands:
- `add`, `change`, `init`, `install`, `list`, `prune`, `remove`, `scan`, `test`, `update`, `version`
- Built-in help: `help`
- Code: `cmd/mmm/*`
- User docs: `docs/commands/`
- Behavior specs: `docs/specs/`

### Configuration files

- `modlist.json` and `modlist-lock.json`
- User guide: `README.md`
- Behavior specs: `docs/specs/`

Additional config surfaces:
- `.mmmignore` rules that affect `scan`, `install`, and `prune`
- User guide: `README.md`

### Output and prompts

- Output and message styles: `internal/output/` and `internal/tui/`
- Translations: `internal/i18n/`

## Interaction components in code

### Execution tier gating

TTY and prompt gating:
- `internal/tui/terminal.go`
- `internal/tui/modes.go`

Global flags that affect interaction:
- `--non-interactive`
- `--quiet`
- `--debug`
- `--config`
- `--perf`
- `--perf-out-dir`
- `--help`

Environment variables that affect what users see:
- `MMM_DISABLE_TELEMETRY`

### Per-command interactive behavior

Commands that can launch Bubble Tea flows:
- `add` recovery state machine: `cmd/mmm/add/tui.go`
- `init` wizard: `cmd/mmm/init/tui.go`
- `install` log wrapper when allowed: `cmd/mmm/install/install.go`
- `list` wrapper when allowed: `cmd/mmm/list/tui.go`
- `change` log wrapper when allowed: `cmd/mmm/change/change.go`
- `test` log wrapper when allowed: `cmd/mmm/test/test.go`
- `update` log wrapper when allowed: `cmd/mmm/update/update.go`

Commands that use simple line prompts:
- `scan` confirmation prompts: `cmd/mmm/scan/scan.go`
- `init` overwrite prompt (non-TUI): `cmd/mmm/init/README.md`
- `prune` deletion confirmation: `cmd/mmm/prune/prune.go`

Commands that are currently non-interactive only:
- `remove`: `cmd/mmm/remove/remove.go`
- `version`: `cmd/mmm/version/version.go`

## Known requirement in progress

Full root-level TUI is a future requirement.
The current implementation prints help when you run `mmm` with no arguments.
See `docs/tui-design-doc.md` for the target model.
