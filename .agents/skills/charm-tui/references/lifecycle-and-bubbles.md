# Component lifecycle and native Bubbles integration

Read [component design](component-design.md) for ownership and
[input routing](input-routing.md) when integrating widget input. For cancellation,
request identity, or results arriving after replacement, also read
[effects and ordering](effects-and-ordering.md).

## Lifecycle is part of the component contract

| Phase | Owning responsibility | Observable contract |
| --- | --- | --- |
| Construction | Component constructor / story factory | Fresh state and explicit inputs; no I/O or ticking; persisted initialization only when explicitly injected |
| Allocation | Parent computes outer rectangle; component computes internal content | Nonnegative dimensions; no duplicate frame subtraction |
| Activation | Parent invokes activation once for this lifetime | Return startup commands and record active identity |
| Focus | Parent chooses recipient; component reflects it | Preserve focus commands, such as cursor startup; repeated focus must not multiply recurring work |
| Update | Root routes; component maintains its local invariant | Only the intended owner changes; returned commands propagate |
| Render | Component produces content; parent composes metadata | Repeatable result for the same explicit inputs; no work starts |
| Resize | Parent reallocates affected children | Retain input and selection; preserve any returned command |
| Blur | Parent removes input ownership | In-flight results still reach their live owner; recurring work follows its explicit active/focus policy |
| Disposal | Owning parent invalidates identity and cancels supported work | No obsolete result can affect a replacement; disposed widgets stop rescheduling ticks |

A fresh constructor or factory starts fresh state unless persistence is
explicitly injected. An injected saved query or restored selection can determine
initial values without making all instances share mutable state. Reading that
persistence still belongs in an owning command or supplied input, not hidden
constructor I/O. Stories use controlled initial values so reselecting a story is
repeatable unless the story explicitly demonstrates persistence.

For a native Bubbles widget, retain its returned model and command together:
`root.input, cmd = root.input.Update(message)`. Its cursor or spinner messages may
need delivery while other user input is routed elsewhere. Read that widget's
pinned source to identify its message filtering and tick scheduling. Do not add a
blanket `if !focused { return }` around all messages.

For example, these root method excerpts use one `textinput.Model`, constructed
with `textinput.New()` from `charm.land/bubbles/v2/textinput`. The caller blurs the
previous focus owner and calls `focusInput` only when focus changes. It returns
the resulting command to Bubble Tea instead of invoking or discarding it.

```go
func (root *rootModel) focusInput() tea.Cmd {
    root.focus = focusEditor
    return root.input.Focus()
}

func (root *rootModel) updateInput(message tea.Msg) tea.Cmd {
    var cmd tea.Cmd
    root.input, cmd = root.input.Update(message)
    return cmd
}
```

Within this single-input root's update switch, the input routes are:

```go
case tea.KeyPressMsg, tea.PasteMsg:
    if root.modal != nil {
        return root.handleModalInput(message)
    }
    if root.focus == focusEditor {
        return root, root.updateInput(message)
    }
    return root.handleOtherInput(message)
default:
    // All application results and other widget routes were handled above.
    // This root has one textinput that owns remaining library messages.
    return root, root.updateInput(message)
```

The fallback includes private cursor-start and clipboard-result message types
that application code cannot name in a type switch. It is a scoped route for
this one input after other owners' messages have been consumed, not a broadcast
loop. `handleModalInput` and `handleOtherInput` are application methods returning
`(tea.Model, tea.Cmd)`. A multi-editor parent needs an explicit clipboard request
owner; delivery to whichever editor happens to be focused on completion can paste
into the wrong field.

Call `root.input.Blur()` when transferring its focus. The widget itself ignores
updates when blurred; its cursor checks blink identity and schedules subsequent
ticks. Preserving that library behavior is different from suppressing every
non-key message in the parent. These details follow the
[textinput implementation](https://github.com/charmbracelet/bubbles/blob/main/textinput/textinput.go)
and [cursor implementation](https://github.com/charmbracelet/bubbles/blob/main/cursor/cursor.go).
For real cursor presentation, also carry `input.Cursor()` into the composed view
as described in [layout and style](layout-and-style.md).

