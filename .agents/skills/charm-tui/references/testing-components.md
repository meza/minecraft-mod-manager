# Component contract tests

Read this when designing, building, reviewing, or testing component behavior,
isolation, focus, or style overrides.

If you have not selected the applicable test layers for the current task, read
[the testing guide](testing.md) first. Read every applicable guide it selects.

Before using or reviewing the Results test example, read the
[worked component](results-example.md), including its input, style, and ownership
assumptions, if you have not already read it.

## Test behavior through the component contract

This test uses the `Results` component contract from the
[worked component](results-example.md): `NewResults`, `SetItems`, `Move`, and
`Selected`. Put it in the
component package's external test package and use its actual module path. The
example path below is a placeholder for that project import.

```go
package results_test

import (
    "testing"

    "example.com/project/internal/ui/results"
)

func TestResultsOwnsItemsAndClampsSelection(t *testing.T) {
    input := []results.Item{
        {ID: "alpha", Label: "Alpha"},
        {ID: "beta", Label: "Beta"},
    }
    first := results.NewResults(input, results.ResultsStyles{})
    second := results.NewResults(input, results.ResultsStyles{})
    input[0].Label = "changed by caller"

    first.Move(100)
    selected, exists := first.Selected()
    if !exists || selected.ID != "beta" {
        t.Fatalf("selection = %#v, %v; want beta", selected, exists)
    }
    selected, exists = second.Selected()
    if !exists || selected.Label != "Alpha" {
        t.Fatalf("fresh instance changed: %#v, %v", selected, exists)
    }

    first.SetItems(nil)
    first.Move(-1)
    if selected, exists := first.Selected(); exists {
        t.Fatalf("empty results selected %#v", selected)
    }
}
```

The test combines a meaningful journey: caller mutation cannot alter owned
items, moving one instance cannot move another, movement stops at the end, and
empty results have no selection. It never inspects a private index or duplicates
the clamping implementation.

For focused controls, test repeated focus and blur through their public methods.
Assert the defined outcome and returned commands rather than requiring all
components to implement an invented common testing interface.

Test style overrides as a behavior invariant. Create fresh instances with the
same data and dependencies but different semantic styles, then apply the same
operations. Assert identical selection, validation, emitted actions, and effect
results. Styles may change appearance and, where the component supports it,
decoration geometry; they must not change domain behavior. Compare geometry
against each style's own allocation contract rather than requiring identical
pixels across themes.
