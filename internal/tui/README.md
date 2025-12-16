# internal/tui

This package is the shared toolbox for Bubble Tea based TUIs in this repo. It does not implement a command's UI by itself; it provides the building blocks so each command can stay consistent.

If you are implementing or changing an interactive command, this package is usually where the shared pieces belong (styles, keymaps, terminal detection).

## Terminal detection (when to launch TUI)

The CLI tries hard to avoid "half a TUI" when input/output are not terminals (CI, pipes, redirected output).

- `ShouldUseTUI(quiet bool, in io.Reader, out io.Writer) bool`
- `ProgramOptions(in io.Reader, out io.Writer) []tea.ProgramOption`

`ProgramOptions` disables Bubble Tea's renderer when no terminal is present.

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

For the architectural expectations of TUIs in this repo, see `docs/tui-design-doc.md` and `docs/tui-guidelines.md`.

## Tests

Run from the repo root:

```bash
make test
```

