# Results component example

Read [component design](component-design.md) for the ownership contract. For
integration, read [input routing](input-routing.md); for asynchronous population,
read [effects and ordering](effects-and-ordering.md). Follow all applicable routes
in component design when building or reviewing a component.

## Worked component: a bounded results selector

This complete component file illustrates a small selector, not a replacement for
the filtering, pagination, help, and delegates already supplied by Bubbles list.
Use Bubbles list when those features are required. This example's narrower
contract needs only replacement, movement, selection, and bounded rendering.

The component copies the input slice. `Item` contains immutable string values, so
a shallow slice copy is sufficient. If items later contain mutable maps or
slices, define ownership or copy those fields too. Keep the component pointer
owned by its parent rather than copying a live component between parents.

`SetItems` preserves the selected position and clamps it when the collection
shrinks; it does not preserve identity across sorting. If the product needs that
behavior, look up the old selected ID in the new collection inside `SetItems`.
Empty selection is represented by `Selected` returning `false`.

The code requires a Go toolchain with `slices.Clone`, `min`, and `max`. Use the
application's pinned v2 dependency versions. Labels are plain single-line text
without terminal control sequences. These style roles carry color and emphasis
only; a surrounding component owns borders and padding.

```go
package results

import (
    "slices"
    "strings"

    "charm.land/lipgloss/v2"
    "github.com/charmbracelet/x/ansi"
)

type Item struct {
    ID    string
    Label string
}

type ResultsStyles struct {
    Normal   lipgloss.Style
    Selected lipgloss.Style
    Empty    lipgloss.Style
}

type Results struct {
    items    []Item
    selected int
    styles   ResultsStyles
}

func NewResults(items []Item, styles ResultsStyles) *Results {
    component := &Results{styles: styles}
    component.SetItems(items)
    return component
}

func (component *Results) SetItems(items []Item) {
    component.items = slices.Clone(items)
    component.selected = min(component.selected, max(0, len(items)-1))
}

func (component *Results) Move(delta int) {
    if len(component.items) == 0 {
        return
    }
    // Clamp the delta first, so even extreme caller values cannot overflow.
    movement := max(-component.selected,
        min(delta, len(component.items)-1-component.selected))
    component.selected += movement
}

func (component *Results) Selected() (Item, bool) {
    if len(component.items) == 0 {
        return Item{}, false
    }
    return component.items[component.selected], true
}

func (component *Results) Render(width, height int) string {
    if width <= 0 || height <= 0 {
        return ""
    }
    if len(component.items) == 0 {
        return ansi.Truncate(component.styles.Empty.Render("No results"), width, "…")
    }

    // Center the selection where possible, while keeping the last page full.
    start := max(0, component.selected-height/2)
    start = min(start, max(0, len(component.items)-height))
    count := min(height, len(component.items)-start)
    lines := make([]string, 0, count)
    for index := start; index < start+count; index++ {
        marker := "  "
        style := component.styles.Normal
        if index == component.selected {
            marker = "> "
            style = component.styles.Selected
        }
        line := style.Render(marker + component.items[index].Label)
        lines = append(lines, ansi.Truncate(line, width, "…"))
    }
    return strings.Join(lines, "\n")
}
```

For two items, `Move(1)` selects the second, `Move(1)` again stays there, and
`SetItems(nil)` makes `Selected()` return `false`. At height one, rendering shows
the selected row. At nonpositive width or height it returns an empty string and
retains the selection for the next usable allocation. The marker makes selection
understandable without relying on color.

The ANSI helper accounts for terminal cells and styling sequences; byte slicing
does not. See [ANSI truncation source](https://github.com/charmbracelet/x/blob/main/ansi/truncate.go)
and the companion [layout reference](layout-and-style.md) for frames, cursor
coordinates, and richer text boundaries.

