# Minecraft Mod Manager TUI Design Recommendations

## Developer Guide

You build one Bubble Tea app per command, and you make it purpose-built for that command's flow (not a reusable "prompt framework"). You model the flow as a finite state machine (FSM) and you lock the UX down with black-box snapshot tests.

### Build TUIs as command apps

- Keep the Bubble Tea model in the command package (example: `cmd/mmm/add/tui.go`).
- Avoid generalized abstractions (generic wizards, generic prompts, generic screens). This project is an application, not a TUI library.
- Keep UI concerns (state, layout, key handling) separate from business logic (API calls, config, file system changes) by injecting the command functions the TUI needs to call (example: `addTUIFetchCmd`).

### Model the flow as an explicit FSM

- Define an enum-like state type (example: `type addTUIState int`) and a constant per screen.
- Centralize transitions in a single place (example: `enterState(state)`), so each state owns its own prompt text, default values, and bubble configuration.
- Track a small history stack so Back is deterministic and easy to reason about (example: `addTUIHistory`).

### Back vs abort

- Treat `Esc` as "Back" (go up one step). Handle it at the top of `Update` before forwarding events to list/input bubbles.
- Treat `Ctrl+C` as "Abort" (quit the entire flow immediately).
- If there is no previous state, `Esc` aborts (you cannot go "up" from the first screen).

### Reuse the shared keybinds, styles, and i18n

- Do not hardcode English text. Use `internal/i18n` (`i18n.T(...)`) for all user-facing strings so snapshots remain stable under `MMM_TEST=true`.
- Use `internal/tui` styles (`tui.QuestionStyle`, `tui.TitleStyle`, `tui.ItemStyle`, `tui.SelectedItemStyle`, `tui.ErrorStyle`) instead of inventing per-command styling.
- Use the translated keymaps (example: `tui.TranslatedListKeyMap()`), and let Bubble Tea render the help footer rather than writing your own.

### Snapshot test every state (black box)

Snapshot tests are your regression harness for UX. They should treat the TUI as a user does: drive it via key presses, wait for the expected output to appear, and snapshot the rendered view.

- Always set `MMM_TEST=true` in snapshot tests so i18n returns stable "key + vars" output (instead of translated text).
- Use `github.com/charmbracelet/x/exp/teatest` to run the model and send key messages.
- Snapshot `model.View()` (or stdout if you are testing the whole command wrapper) using go-snaps.
- To update snapshots, follow the instructions in `CONTRIBUTING.md`.

Example (from `cmd/mmm/add/tui_test.go`):

```go
t.Setenv("MMM_TEST", "true")
tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(60, 20))
tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
snaps.MatchSnapshot(t, final.View())
```

### TTY behavior

Use `internal/tui.ProgramOptions(in, out)` when you construct a Bubble Tea program so it disables the renderer when stdin/stdout are not TTYs. For cross-package tests that need deterministic TTY behavior, override terminal detection via `tui.SetIsTerminalFuncForTesting(...)` and restore it afterwards.

## Modern Terminal UI Principles

### Layout

Modern terminal UIs (TUIs) favour a **multi‑panel dashboard** rather than a single scrolling view. A typical arrangement consists of a sidebar for navigation, a main panel for content and a footer for status information.

Widgets live inside container “boxes” that control padding, margins, borders and alignment, much like CSS. With [Lip Gloss](https://github.com/charmbracelet/lipgloss) you can compose panels using functions such as `JoinHorizontal`, `JoinVertical` or by simply concatenating styled strings.

```go
sidebarStyle := lipgloss.NewStyle().
    Padding(1, 2).
    Border(true, false, false, false) // left, top, right, bottom
```

### Keyboard Navigation

TUIs are designed for keyboard use only. Up/Down arrow keys (or **k / j** for Vim fans) move between items, while **Enter** or **Space** confirms a choice. Press **q**, **Esc** or **Ctrl+C** to exit or step back. Keeping a footer with these shortcuts makes the controls self‑documenting:

```
↑/↓ Navigate • ↵ Select • q Quit • h Help
```

### Colour & Theme

A dark theme works best for remote SSH sessions: avoid pure black or white and favour charcoal greys behind off‑white text. Lip Gloss’ adaptive colours mean you can specify a single logical colour that switches automatically between light and dark terminals:

```go
lipgloss.AdaptiveColor{Light: "#bbbbbb", Dark: "#888888"}
```

Restrict yourself to one or two bright accent colours for selections, progress bars and other highlights so the palette remains tidy.

### Responsive Layout

Listen for `tea.WindowSizeMsg` and recalculate panel sizes whenever the terminal is resized:

```go
case tea.WindowSizeMsg:
    m.width, m.height = msg.Width, msg.Height
    sidebar.SetSize(m.height, 18)           // fixed sidebar width
    main.SetSize(m.width-20, m.height-2)    // remaining width
```

When the programme starts over SSH, sending a delayed `WindowSizeMsg` forces the UI to snap to the client’s size.

### Input & Interaction

The [Bubbles](https://github.com/charmbracelet/bubbles) library covers most interaction patterns out of the box. Use lists or tables for selection, text inputs for user entry, viewports for scrollable logs and spinners or progress bars for long‑running tasks. Always show a visible cursor or focus indicator and consider adding contextual help via `list.Help()`.

## Common UX Conventions

Good TUIs share a few habits: intuitive navigation (Tab or arrow keys to switch focus), explicit feedback (spinners while work is in progress and a message when it completes), quick help (key‑bindings in the footer or a pop‑up toggled with **h**) and a clear visual hierarchy created with padding, borders and bold section titles.

---

## Recommended Layout for Minecraft Mod Manager

A compact three‑pane layout works well:

```
+----------------------+-------------------------------------------+
| [Add   ]             |                                           |
|  Remove              |   [Add Mod]                               |
|  Update              |   > Enter Modrinth/CurseForge ID: [____]  |
|  Scan                |   [Press Enter to add or Esc to cancel]   |
|  List                |                                           |
|  Prune               |                                           |
|  Test                |                                           |
|  Change version      |                                           |
+----------------------+-------------------------------------------+
| ↑/↓ Navigate • ↵ Select • Q Quit • H Help                |
+---------------------------------------------------------+
```

### Sidebar (Commands)

The sidebar is a fixed‑width column of primary commands (about 18 characters wide). Highlight the focused row with inverted colours or bold text.

```go
sidebarStyle := lipgloss.NewStyle().
    Padding(1, 2).
    Width(18).
    Border(lipgloss.NormalBorder(), true, false, true, false).
    BorderForeground(lipgloss.Color("240")).
    Background(lipgloss.Color("235"))
```

### Main Panel (Context‑Sensitive)

The main panel changes with the selected command:

* **Add** — prompt for a Modrinth/CurseForge ID via a `TextInput` bubble.
* **Remove** — show installed mods in a selectable list and allow multiple deletion.
* **Update** — display a table of installed mods against their latest versions with an option to update one or all.
* **Scan** — list unmanaged JARs found in the `mods/` folder and let the user import selected files.
* **List** — present a read‑only table of installed mods.

### Footer (Key Hints)

A single persistent line with a dim background keeps essential shortcuts visible:

```go
footerStyle := lipgloss.NewStyle().
    Foreground(lipgloss.Color("250")).
    Background(lipgloss.Color("236"))
```

Display something like: `↑/↓ Navigate • ↵ Select • H Help • Q Quit`.

---

## Implementation Notes (Bubble Tea & Lip Gloss)

Bubble Tea applications revolve around a model that handles updates and view rendering. Use `list.Model` for the sidebar and switch the active main view inside the update function. A short sleep followed by a resize command on start‑up ensures the UI initialises at the correct size.

Reusable styles make the code easier to read:

```go
titleStyle := lipgloss.NewStyle().
    Bold(true).
    Foreground(lipgloss.Color("#FAFAFA"))

mainStyle := lipgloss.NewStyle().
    Padding(1, 2)
```

Compose panels with `lipgloss.JoinHorizontal(lipgloss.Top, sidebarBox, mainBox)` and align content using `PlaceHorizontal` or `PlaceVertical`. For consistency, define a key map so that **q** and **Ctrl+C** always quit (`tea.Quit`) and **h** toggles help.

---

## Overall Guidance

Aim for a clean, compact aesthetic with minimal borders, consistent padding and tidy monospaced alignment. Reducing blank lines keeps the interface concise yet readable, which is particularly important when users connect over SSH.

---

## Sources & Further Reading

* Our TUI Design Doc — [docs/tui-design-doc.md](./tui-design-doc.md)
* Bubble Tea — [https://github.com/charmbracelet/bubbletea](https://github.com/charmbracelet/bubbletea)
* Lip Gloss — [https://github.com/charmbracelet/lipgloss](https://github.com/charmbracelet/lipgloss)
* Bubbles — [https://pkg.go.dev/github.com/charmbracelet/bubbles](https://pkg.go.dev/github.com/charmbracelet/bubbles)
* Real Python: **Python Textual – Build Beautiful UIs in the Terminal** — [https://realpython.com/python-textual/](https://realpython.com/python-textual/)
* Teatutor Deep Dive — [https://zackproser.com/blog/teatutor-deepdive](https://zackproser.com/blog/teatutor-deepdive)
* Bubble Tea guide (Obsidian vault) — [https://publish.obsidian.md/manuel/Wiki/Programming/bubbletea](https://publish.obsidian.md/manuel/Wiki/Programming/bubbletea)
