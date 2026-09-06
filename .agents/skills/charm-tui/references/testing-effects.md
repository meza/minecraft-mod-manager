# Effect and time tests

Read this when designing, building, reviewing, or testing commands, asynchronous
results, loading, retries, cancellation, or time-dependent behavior.

If you have not selected the applicable test layers for the current task, read
[the testing guide](testing.md) first. Read every applicable guide it selects.

## Test a bounded command and its result

The following complete test-file example covers both the effect boundary and
the full `Update` to command to result to `Update` transition. Its small
application root owns a search request and uses an injected lookup function.
Reusable leaves still expose narrow operations; this root implements the native
Bubble Tea model contract.

```go
package search_test

import (
    "strings"
    "testing"

    tea "charm.land/bubbletea/v2"
)

type searchLoaded struct {
    owner   uint64
    request uint64
    query   string
    labels  []string
    err     error
}

func loadSearch(
    owner uint64, request uint64, query string,
    lookup func(string) ([]string, error),
) tea.Cmd {
    return func() tea.Msg {
        labels, err := lookup(query)
        return searchLoaded{
            owner: owner, request: request, query: query,
            labels: append([]string(nil), labels...), err: err,
        }
    }
}

type submitSearch struct{ query string }

type searchRoot struct {
    owner   uint64
    request uint64
    loading bool
    labels  []string
    err     error
    lookup  func(string) ([]string, error)
}

func (root searchRoot) Init() tea.Cmd { return nil }

func (root searchRoot) Update(message tea.Msg) (tea.Model, tea.Cmd) {
    switch message := message.(type) {
    case submitSearch:
        root.request++
        root.loading = true
        root.labels = nil
        root.err = nil
        return root, loadSearch(root.owner, root.request, message.query, root.lookup)
    case searchLoaded:
        if message.owner != root.owner || message.request != root.request {
            return root, nil
        }
        root.loading = false
        root.err = message.err
        if message.err == nil {
            root.labels = append([]string(nil), message.labels...)
        }
    }
    return root, nil
}

func (root searchRoot) View() tea.View {
    if root.loading {
        return tea.NewView("Loading")
    }
    if root.err != nil {
        return tea.NewView("Search failed")
    }
    return tea.NewView(strings.Join(root.labels, "\n"))
}

func TestSearchSubmitLoadsThroughUpdate(t *testing.T) {
    root := searchRoot{
        owner: 7,
        lookup: func(query string) ([]string, error) {
            if query != "cha" {
                t.Fatalf("query = %q, want cha", query)
            }
            return []string{"Charm"}, nil
        },
    }

    next, cmd := root.Update(submitSearch{query: "cha"})
    loading := next.(searchRoot)
    if !loading.loading || loading.View().Content != "Loading" {
        t.Fatal("submit did not enter loading state")
    }
    if cmd == nil {
        t.Fatal("submit did not return a load command")
    }

    message := cmd()
    if !loading.loading || len(loading.labels) != 0 {
        t.Fatal("command changed UI state before Update received its result")
    }
    next, _ = loading.Update(message)
    loaded := next.(searchRoot)
    if loaded.loading || loaded.err != nil {
        t.Fatalf("load did not finish successfully: %#v", loaded)
    }
    if len(loaded.labels) != 1 || loaded.labels[0] != "Charm" {
        t.Fatalf("labels = %#v, want Charm", loaded.labels)
    }
    if loaded.View().Content != "Charm" {
        t.Fatalf("view = %q, want Charm", loaded.View().Content)
    }
}

func TestLoadSearchCapturesRequestAndDefersLookup(t *testing.T) {
    calls := 0
    query := "cha"
    cmd := loadSearch(7, 2, query, func(received string) ([]string, error) {
        calls++
        if received != "cha" {
            t.Fatalf("query = %q, want cha", received)
        }
        return []string{"Charm"}, nil
    })
    query = "changed"
    if calls != 0 {
        t.Fatal("lookup ran while creating the command")
    }

    message, ok := cmd().(searchLoaded)
    if !ok || message.owner != 7 || message.request != 2 || message.query != "cha" {
        t.Fatalf("unexpected result: %#v", message)
    }
    if calls != 1 || message.err != nil || len(message.labels) != 1 {
        t.Fatalf("lookup calls = %d, result = %#v", calls, message)
    }
}
```

The lookup runs synchronously only because this test deliberately invokes a
bounded fake. In the application, Bubble Tea runs
the command. Add a failing lookup case using a sentinel error and assert that
the result retains both identity and error.

Apply this same transition test to the project's real root with its controlled
dependencies, including an error response and retry. Deliver an older request
after a newer result and assert no change. Deliver a replaced owner's result and
assert it cannot reach the new instance. Cancellation alone cannot establish
either guarantee.

Do not recursively execute arbitrary commands in a homemade test runner.
`tea.Batch` and `tea.Sequence` involve runtime scheduling, and a real tick or
network command can wait indefinitely. Test a specific bounded command directly;
use runtime integration when scheduling is the subject of the test. Fake the
application boundary with `httptest.NewServer`, `t.TempDir`, environment overrides,
or an injected function, rather than mocking Bubble Tea's event loop. Use
environment overrides when testing configuration that the application actually
reads from its environment; restore the environment after the test so another
case does not inherit it. Fixed timestamps provide the corresponding time seam.

## Control time explicitly

Inject the clock when a state decision reads time. Inject a tick producer when
the effect itself waits. Replacing `time.Now` alone does not control a real
`tea.Tick` or timer.

For example, a refresh rule can depend on `now func() time.Time`. A direct test
changes the captured timestamp between calls:

```go
currentTime := time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)
now := func() time.Time { return currentTime }
lastRefresh := now()
currentTime = currentTime.Add(2 * time.Second)
if now().Sub(lastRefresh) < time.Second {
    t.Fatal("expected refresh interval to have elapsed")
}
```

This illustrates the clock seam; in a component test, assert the component's
refresh decision after advancing `currentTime`. For debounce and retry, deliver
the named tick/result message with its request identity and test both current
and obsolete identities. Use harness timeouts to fail a hung test, not sleeps to
make application behavior probably happen.

