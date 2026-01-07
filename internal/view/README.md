# internal/view

This package is the shared toolbox for Bubble Tea based views in this repo. It does not implement a command's UI by itself; it provides the building blocks so each command can stay consistent.

If you are implementing or changing an interactive command, this package is usually where the shared pieces belong (styles, keymaps, terminal detection).

## View capabilities

The CLI tries hard to avoid "half a view" when input/output are not terminals (CI, pipes, redirected output).

- `SupportsPrompting(in io.Reader, out io.Writer) bool`
- `SupportsColor(writer io.Writer) bool`
- `SupportsUnicode() bool`
- `SupportsControlSequences(out io.Writer) bool`
- `ProgramOptions(in io.Reader, out io.Writer) []tea.ProgramOption`

`ProgramOptions` disables Bubble Tea's renderer when output cannot use control sequences.
Commands decide how `--unattended` affects prompting; the view package only reports capabilities.

`SupportsColor` reports whether the environment supports color, and `SupportsUnicode` reports whether Unicode output is supported. Neither implies that control sequences are safe to emit; gate ANSI styling on `SupportsControlSequences`.

For tests that need deterministic behavior across platforms:

- `SetIsTerminalFuncForTesting(fn func(int) bool) func()` returns a restore function

## Shared styles

`styling.go` contains shared Lip Gloss styles used across prompts and lists. Commands import these styles so the UI feels cohesive.

## Shared keymaps

`keybindings.go` and `keymaps.go` wrap Charm's key helpers with translated help text:

- `TranslatedInputKeyMap` for text inputs
- `TranslatedListKeyMap()` for list views

These are intentionally thin wrappers so key help stays consistent and localizable.

## Related docs

For terminal interaction expectations and implementation guidance, see:
- `docs/interactions/interaction-guidelines.md`
- `docs/guide-to-working-with-the-terminal.md`

## Tests

See `CONTRIBUTING.md` for required test/coverage checks and snapshot update instructions.
