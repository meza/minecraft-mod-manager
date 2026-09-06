# Layout, styles and cursor coordinates

Use this reference when implementing or reviewing rendering, resize behaviour,
themes or real terminal cursors. The parent assigns an outer rectangle; the child
owns everything inside it. Keep the same contract in the application and in
[Bubblebook](bubblebook.md).

## Inject meaning, allocate space separately

Resolve the palette at the application boundary and pass ordinary style values
to components. A selected row owns its selected appearance; its parent decides
how much screen space the component receives. A theme change replaces styles and
recomputes geometry if the new decoration differs.

The research contrasts hard-coded tokens inside a reusable component with a
factory that maps the application's semantic theme into component styles.
This package-global style fixes every host to one selected color:

```go
// Incorrect boundary for a reusable component's host-controlled palette.
var selectedStyle = lipgloss.NewStyle().
    Foreground(lipgloss.Color("205"))
```

Instead, use the research's parameterized style factory:

```go
type Styles struct {
    Normal   lipgloss.Style
    Selected lipgloss.Style
    Error    lipgloss.Style
}

func DefaultStyles(theme Theme) Styles {
    return Styles{
        Normal:   lipgloss.NewStyle().Foreground(theme.Text),
        Selected: lipgloss.NewStyle().Foreground(theme.Accent).Bold(true),
        Error:    lipgloss.NewStyle().Foreground(theme.Error),
    }
}
```

This is a Go excerpt using `charm.land/lipgloss/v2`. `Theme` is the application's
semantic token type; its `Text`, `Accent`, and `Error` fields provide colors
accepted by `Foreground`. The application supplies its own theme, and Bubblebook
can render the component under multiple themes through the same factory. These
component-specific `Styles` are separate from the frame example below. Fixed
palette literals belong at the application's palette boundary or in controlled
fixtures, rather than inside the reusable component's styling policy.

This Go excerpt defines a separate decorated-row example and one palette. It uses
`charm.land/lipgloss/v2`. These names are application code, not framework APIs.
This example is independent of the unframed `Results` component in the
[worked component reference](results-example.md); it demonstrates a component that owns its frame.
The row styles intentionally have no width, padding or border: the frame owns
decoration. Selection also gets a textual marker in the row renderer.

```go
type FramedRowStyles struct {
    Normal   lipgloss.Style
    Selected lipgloss.Style
    Empty    lipgloss.Style
    Frame    lipgloss.Style
}

func darkFramedRowStyles() FramedRowStyles {
    return FramedRowStyles{
        Normal:   lipgloss.NewStyle().Foreground(lipgloss.Color("#E6E6E6")),
        Selected: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F2C94C")),
        Empty:    lipgloss.NewStyle().Foreground(lipgloss.Color("#A0A0A0")),
        Frame: lipgloss.NewStyle().
            Border(lipgloss.RoundedBorder()).
            BorderForeground(lipgloss.Color("#777777")).
            Padding(0, 1),
    }
}
```

| Input | Owner | Example |
| --- | --- | --- |
| Palette and theme choice | Application boundary | Light or dark resolved colors |
| Semantic role | Component style contract | Selected, empty, error |
| Outer rectangle | Parent layout | Results receives 46 columns by 12 rows |
| Decoration and content rectangle | Component | Border plus one cell of horizontal padding |
| Domain state | Its authoritative state owner | Selected result ID survives theme changes |

Avoid terminal probing inside a renderer and mutable package-level component
styles. Both create hidden inputs that make two stories or tests influence one
another. Lip Gloss supplies value-based style construction and rendering; inspect
the project's pinned implementation when changing its settings.
[Lip Gloss style implementation](https://github.com/charmbracelet/lipgloss/blob/main/style.go)

## Work from the outside in

For a child allocated 46 columns and 12 rows, with a one-cell border on all sides
and horizontal padding of one cell per side:

```text
Parent allocation: 46 columns x 12 rows
┌────────────────────────────────────────────┐  border: 1 row
│ [content: 42 columns x 10 rows]             │
│                                            │
└────────────────────────────────────────────┘  border: 1 row
  ^                                        ^
  horizontal padding: 1 cell on each side

Horizontal frame = 1 border + 1 padding + 1 padding + 1 border = 4
Vertical frame   = 1 border + 1 border = 2
Content          = (46 - 4) x (12 - 2) = 42 x 10
Content origin   = outer origin + (2, 1)
```

The drawing is schematic; the arithmetic defines the allocation. Margins are
outside the border but still consume the component's allocated rectangle. Reserve
parent-owned gaps in the parent instead of charging the child for them twice.

This allocation excerpt uses actual style getters, including margins. Call it
again when the frame style changes, not only when the terminal changes size.

```go
contentWidth := max(0, outerWidth-frame.GetHorizontalFrameSize())
contentHeight := max(0, outerHeight-frame.GetVerticalFrameSize())
contentLeft := frame.GetMarginLeft() + frame.GetBorderLeftSize() + frame.GetPaddingLeft()
contentTop := frame.GetMarginTop() + frame.GetBorderTopSize() + frame.GetPaddingTop()
```

These methods account for the enabled sides of the actual frame. A guessed
`width - 4` stops being correct when a theme changes padding or a border disappears.
[Lip Gloss frame getters](https://github.com/charmbracelet/lipgloss/blob/main/get.go)

Do not infer clipping from `Width`. Lip Gloss v2's rendering calculation includes
borders in its width setting and then handles padding; margins are separate.
Passing an already reduced content width as the frame width can subtract space
again. Either render already bounded content with a frame that has no dimensions,
as below, or establish the pinned style API's exact width semantics before using
its sizing settings. Measure the final output as well as the content.
[Lip Gloss rendering calculation](https://github.com/charmbracelet/lipgloss/blob/main/style.go)

## Bound a row before adding decoration

The following complete function is an instructional single-row renderer, not a
replacement for a Bubbles list. Its imports are `strings`,
`charm.land/lipgloss/v2` and `github.com/charmbracelet/x/ansi`. Its input is one
display line: no tabs, newlines, cursor movement or other terminal control commands;
SGR styling is allowed. The supplied frame has only ordinary borders, nonnegative
padding/margins and colors, with no stored text, transform, dimensions or alignment.

```go
func renderFramedRow(label string, width, height int, frame lipgloss.Style) string {
    if width <= 0 || height <= 0 {
        return ""
    }
    contentWidth := max(0, width-frame.GetHorizontalFrameSize())
    contentHeight := max(0, height-frame.GetVerticalFrameSize())
    if contentWidth == 0 || contentHeight == 0 {
        return "" // This component hides when its frame plus content cannot fit.
    }

    row := ansi.Truncate(label, contentWidth, "…")
    row += strings.Repeat(" ", max(0, contentWidth-ansi.StringWidth(row)))
    return frame.Render(row)
}
```

With the style above, `renderFramedRow("Ready", 12, 3, styles.Frame)` has
8 content cells, 12 outer columns and 3 outer rows. It uses less than the available
height when passed a taller allocation; the parent owns any remaining space.
At width 4 or height 2 it returns an empty string. It preserves the label in state,
so growing the terminal reveals it again.

For multi-row content, select visible rows or wrap at the content width, enforce
the content height, then add the frame once. An editor should scroll to keep its
insertion point visible using the widget's own viewport behaviour; truncating its
rendered string independently can invalidate its cursor coordinates.

| Case | Required result |
| --- | --- |
| Zero width or height | Empty output; no cursor |
| Frame cannot leave a content cell | This contract hides the component |
| Long label | ANSI-aware truncation fits the content width |
| Too many rows | Component viewport limits rows before decoration |
| Prompt or scrollbar | Subtract its actual occupied cells inside the content area |
| Theme changes frame size | Reallocate content and update cursor offsets |
| Shrink then grow | Query, items and selection remain intact |

`len` counts bytes; it cannot measure terminal columns. For example, `界` occupies
two cells under the chosen width model, combining marks belong to their grapheme,
and SGR sequences add no cells. Use `ansi.StringWidth` with `ansi.Truncate` for this
path. `ansi.Truncate(label, 5, "…")` reserves the tail inside its five-cell budget;
do not add another ellipsis afterward. Pin the width library and terminal assumptions
in render tests, especially for emoji and ambiguous-width characters.
[ANSI truncation implementation](https://github.com/charmbracelet/x/blob/main/ansi/truncate.go)

## Translate a cursor at each composition boundary

### Case study: replace hand-tuned cursor offsets

The research identifies [Crush PR #2530](https://github.com/charmbracelet/crush/pull/2530)
as a concrete example. Onboarding cursor positioning used hand-tuned coordinate
adjustments. The merged fix derived positioning from the actual dialog style
geometry and removed duplicated special cases.

The research illustrates the bad pattern with:

```go
cursor.X -= 1
cursor.Y -= 1
```

Its corresponding good pattern is:

```go
cursor = AdjustInputCursor(cursor, dialogStyle)
```

`AdjustInputCursor` is the research's illustrative helper name, not a Lip Gloss
or Bubble Tea API. The lesson extends beyond cursors: frame, padding and border
calculations belong in shared helpers whose tests enumerate style variants. The
worked coordinate translation below develops that same principle.

### Compose visible coordinates

A component that renders a real cursor returns its string plus a local cursor.
The local coordinate must describe the actual visible content, including any
prompt and widget scrolling. It is neither a byte index nor automatically the
length of the complete query. Do not derive it again in the parent.

```text
Widget visible cursor (5, 0)
    + prompt origin (3, 0), if not already included by widget
    + component content origin (2, 1)
    = cursor relative to component outer rectangle (10, 1)
    + component placement in root (12, 4)
    = terminal cursor (22, 5)
```

Record which coordinate system the component returns. If it already includes the
prompt, adding the prompt again is a bug. An overlay's centering or a parent's
vertical header contributes an offset in its own composition boundary.

This complete helper excerpt imports the standard `image` package and
`tea "charm.land/bubbletea/v2"`. It copies metadata before translating it, so
rendering cannot mutate the component's cursor. The clip rectangle is in the
destination coordinate system and is the intersection of the visible content
region and the parent's visible allocation, excluding decorations.

```go
func cursorAt(local *tea.Cursor, origin image.Point, clip image.Rectangle) *tea.Cursor {
    if local == nil {
        return nil
    }
    translated := *local
    translated.X += origin.X
    translated.Y += origin.Y
    if !image.Pt(translated.X, translated.Y).In(clip) {
        return nil
    }
    return &translated
}
```

Root `View` integration excerpt: `content` is the fully composed string;
`focusedCursor` belongs to the one focused, visible control, or is nil;
`contentOrigin` is that cursor space's origin in the composed output;
`visibleContent` is its visible clip rectangle. An active modal replaces the
underlying cursor candidate. Apply the terminal bounds as the final clip.

```go
view := tea.NewView(content)
terminalBounds := image.Rect(0, 0, terminalWidth, terminalHeight)
view.Cursor = cursorAt(focusedCursor, contentOrigin, visibleContent.Intersect(terminalBounds))
return view
```

`tea.View.Cursor` carries the runtime cursor; string composition alone does not.
The root also owns terminal policy such as `AltScreen`. A component does not
toggle terminal modes to place its cursor. The Bubblebook adapter preserves the
same local metadata and lets the host translate it into the preview placement.
[Bubble Tea View and Cursor types](https://github.com/charmbracelet/bubbletea/blob/main/tea.go)

## Resize is a state transition with a return path

The root receives `tea.WindowSizeMsg`, allocates outer rectangles, and calls each
affected component's resize operation once. The child derives its own content
area and updates its internal Bubbles widgets. It retains their updated models
and returns any resulting commands to the caller. The root returns those commands
to Bubble Tea. The story host follows the same chain.

This root update excerpt assumes separate editor and details components with
`Resize(width, height int) tea.Cmd`, and already computed allocations. These are
effectful component contracts, distinct from the worked `Results` renderer that
receives its dimensions directly in `Render`.
Both resizes are independent; their synchronous state changes happen before
their returned commands are batched. If one operation depends on the other's
result message, handle that message before starting the dependent operation.

```go
case tea.WindowSizeMsg:
    // Calculate editor and details allocations from this message first.
    editorCmd := root.editor.Resize(editorWidth, editorHeight)
    detailsCmd := root.details.Resize(detailsWidth, detailsHeight)
    return root, tea.Batch(editorCmd, detailsCmd)
```

A pure resize legitimately returns nil. Do not create a command only to change
dimensions, call `Init` again on every resize, or rebuild the component and lose
its editing state. When wrapping a widget's `Update`, assigning the returned
model while discarding its command is an incomplete integration.

Verify geometry through public rendering and resize contracts: compare final
line cell widths and row counts with allocations, check tiny decorated sizes,
change padding and border sides, shrink then restore the allocation, and assert
cursor coordinates and visibility separately from the rendered string. See the
[testing reference](testing.md) for the wider deterministic test workflow.
