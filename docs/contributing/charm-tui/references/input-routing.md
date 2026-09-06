# Input and configuration routing

Read [component design](component-design.md) for the ownership contract and the
[Results example](results-example.md) before using the Results routing excerpt;
it supplies the component referred to below. For focus transitions or native
Bubbles integration, also read [lifecycle and Bubbles](lifecycle-and-bubbles.md).
For routing asynchronous results, also read
[effects and ordering](effects-and-ordering.md).

## Translate input once at its owner

Incorrect: broadcasting each message to every child. This illustrative excerpt
assumes `root.children` contains models and `commands` collects their commands:

```go
for index := range root.children {
    next, cmd := root.children[index].Update(message)
    root.children[index] = next
    commands = append(commands, cmd)
}
```

Every key now reaches every child. Several controls can react to the same
shortcut, a modal cannot reliably suppress underlying input, and each child must
rediscover focus policy. Collecting the returned commands correctly does not fix
the ownership error.

The root instead distinguishes deliberately distributed configuration from
targeted interaction. This routing-outline excerpt uses the application-defined
`ThemeChangedMsg`; its comments identify work performed by the owning root:

```go
switch message := message.(type) {
case tea.WindowSizeMsg:
    // Recompute layout, then resize affected children with their allocations.
case ThemeChangedMsg:
    // Resolve the semantic style set and update affected components.
case tea.KeyPressMsg:
    if root.dialog.Visible() {
        return root.handleDialog(message)
    }
    switch root.focus {
    case focusEditor:
        return root.handleEditor(message)
    case focusResults:
        return root.handleResults(message)
    }
}
```

Here `dialog`, the focus constants, and the three handlers are application
contracts, not methods required of the Results example. Theme updates preserve
component data and interaction state. The root resolves style values once and
passes them down through each component's style configuration contract. Messages
can be globally produced without being globally consumed.

The following is an excerpt from a root `Update`, using the component above as
`root.results`. `root.focus`, `root.modal`, and `root.openItem` are application
coordination contracts, not library APIs. `handleModalKey` returns
`(tea.Model, tea.Cmd)` and cannot forward an unhandled key to underlying controls
while the modal owns input. `openItem` decides the application action for an ID.

```go
case tea.KeyPressMsg:
    if message.String() == "ctrl+c" {
        return root, tea.Quit // This application's explicit global quit policy.
    }
    if root.modal != nil {
        return root.handleModalKey(message)
    }
    if root.focus != focusResults {
        return root, nil
    }
    switch message.String() {
    case "up", "k":
        root.results.Move(-1)
    case "down", "j":
        root.results.Move(1)
    case "enter":
        if item, found := root.results.Selected(); found {
            return root, root.openItem(item.ID)
        }
    }
```

This code belongs inside a type switch binding `message := message.(type)`.
Other focused controls have their own route before the fallback return. Search
results, resize, and animation messages are separate cases outside this key
branch. The exact key types are defined by
[Bubble Tea v2's key API](https://github.com/charmbracelet/bubbletea/blob/main/key.go).

The root should not store another selection index. It chooses key bindings and
application actions; the selector chooses how movement preserves its invariant.
For mouse interaction, the parent hit-tests its actual allocated regions and
converts the event to local coordinates before invoking a component operation.

