# Effects and command ordering

Read [component design](component-design.md) for state ownership and the
[Results example](results-example.md) for the results package used below. For
owner cancellation, replacement, or disposal, also read
[lifecycle and Bubbles](lifecycle-and-bubbles.md). For tasks spanning input and
result routing, also read [input routing](input-routing.md).

## Inject effects and reject obsolete results

Incorrect: performing blocking work in `Update`. This example retains the
research's file-loading mechanism. `rootModel`, `refreshMsg`, `parseRows`, and
the root's `path`, `rows`, and `err` fields are illustrative application contracts;
`os.ReadFile` is the standard-library call:

```go
func (root rootModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
    switch message.(type) {
    case refreshMsg:
        contents, err := os.ReadFile(root.path) // Blocks the event loop.
        if err != nil {
            root.err = err
            return root, nil
        }
        root.rows = parseRows(contents)
    }
    return root, nil
}
```

Moving the call into a command is necessary, but this second example is still
incorrect because the closure retains and mutates the model. `slowCall` and
`finishedMsg` represent the application's operation and completion message:

```go
func (root *rootModel) start() tea.Cmd {
    return func() tea.Msg {
        root.loading = false // Hidden concurrent mutation.
        root.result = slowCall()
        return finishedMsg{}
    }
}
```

The correction is to capture inputs and the injected operation, return a typed
result, and apply it in `Update`. The following search example demonstrates the
same mechanism with cancellation and stale-result handling.

These excerpts belong together in an application package importing `context`,
Bubble Tea as `tea`, and its own `results` package. The screen instance carries an
`ownerID` allocated uniquely by its parent; the parent must not reuse an ID while
old messages can still arrive. The root also owns `requestID`, `cancelSearch`,
`search`, `loading`, `searchError`, and `results`. These are application fields,
not additional methods required of the results component.

```go
type SearchFunc func(context.Context, string) ([]results.Item, error)

type searchLoadedMsg struct {
    ownerID   uint64
    requestID uint64
    items     []results.Item
    err       error
}

func searchCommand(ctx context.Context, search SearchFunc, query string,
    ownerID, requestID uint64) tea.Cmd {
    return func() tea.Msg {
        items, err := search(ctx, query)
        return searchLoadedMsg{
            ownerID: ownerID, requestID: requestID, items: items, err: err,
        }
    }
}

func (root *rootModel) startSearch(ctx context.Context, query string) tea.Cmd {
    if root.cancelSearch != nil {
        root.cancelSearch()
    }
    requestContext, cancel := context.WithCancel(ctx)
    root.cancelSearch = cancel
    root.requestID++
    root.loading = true
    root.searchError = nil
    return searchCommand(requestContext, root.search, query,
        root.ownerID, root.requestID)
}
```

The command closes over an immutable query, identities, context, and injected
function. It never captures the mutable root. The service returns an owned result
slice and does not keep mutating it after return. A test can supply a bounded fake
function; production supplies a service that observes the request context.

Handle the completion in `Update`, independently of focus:

```go
case searchLoadedMsg:
    if message.ownerID != root.ownerID || message.requestID != root.requestID {
        return root, nil
    }
    if root.cancelSearch != nil {
        root.cancelSearch()
        root.cancelSearch = nil
    }
    root.loading = false
    root.searchError = message.err
    if message.err == nil {
        root.results.SetItems(message.items)
    }
    return root, nil
```

Here failure retains the previous results and exposes an error alongside them.
The root's view must label retained data as previous results and show loading or
failure explicitly. Retry calls `startSearch` again with the desired query.
Success with zero items replaces the old collection and renders the empty state.

When leaving the screen, its owner cancels active work and invalidates the live
instance before another screen can accept results. Cancellation reduces wasted
work; identity checks protect correctness even if work completed before the
cancellation or the service did not stop promptly.

Calling the service directly inside `Update` blocks all input and redraws.
Assigning `root.results` inside a command races the event loop. Returning a typed
message avoids both failures and makes failure, retry, and stale completion
behavior independently testable.

## Make dependencies stronger than execution order

`tea.Batch` runs independent commands concurrently; `tea.Sequence` executes them
in order. Neither declaration means a prior operation succeeded. The API contract
is in [Bubble Tea's command source](https://github.com/charmbracelet/bubbletea/blob/main/commands.go).

For independent refresh operations, an application excerpt can return:

```go
return root, tea.Batch(root.refreshAccount(), root.refreshNotifications())
```

Each operation returns its own typed result and has its own live-request check.

### Concurrent pages: arrival order must not become display order

The research's page example starts three independent requests:

```go
return root, tea.Batch(
    requestPage(1),
    requestPage(2),
    requestPage(3),
)
```

Incorrect aggregation in `Update` assumes page one completes before page two:

```go
case pageLoadedMsg:
    root.rows = append(root.rows, message.rows...) // Assumes 1, then 2, then 3.
```

Preserve concurrency while correcting the reducer:

```go
case pageLoadedMsg:
    root.pages[message.page] = message.rows
    root.rows = flattenPagesInOrder(root.pages)
```

These are application excerpts: `requestPage` returns a command;
`pageLoadedMsg` carries the requested page number and rows; `root.pages` is an
initialized map indexed by page number; and `flattenPagesInOrder` concatenates
the available pages in ascending numeric order, never map iteration order.
Initialize a new page map for a new query and reject obsolete request identities
before this aggregation. Handle each page's error through its typed result
contract rather than treating a failed page as a successful empty page.

If pages complete in order 3, 1, 2, appending yields 3, 1, 2; keyed aggregation
ultimately yields 1, 2, 3. Completion order no longer controls visible ordering.

### A success-dependent action starts from the result

For a save followed by closing a dialog, start only the save command. These switch
cases illustrate the application-owned `saveRequestedMsg` and `saveFinishedMsg`
contracts; `saveFinishedMsg` includes `ownerID`, `requestID`, and `err`.

```go
case saveRequestedMsg:
    return root, root.startSave(message.value)
case saveFinishedMsg:
    if !root.isCurrentSave(message.ownerID, message.requestID) {
        return root, nil
    }
    root.saving = false
    if message.err != nil {
        root.saveError = message.err
        return root, nil
    }
    root.closeDialog()
    return root, nil
```

The dialog stays open on failure. Replacing this with
`tea.Sequence(saveCommand, closeCommand)` would run the close even when the save
returns an error message. Use `Sequence` only when execution order itself is the
requirement and the later command does not depend on an updated model or a success
decision. Return to `Update` whenever the next action has such a precondition.

