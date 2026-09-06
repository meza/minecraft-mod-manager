# Runtime integration tests

Read this when designing, building, reviewing, or testing behavior that depends
on the real Bubble Tea runtime delivering commands and events.

If you have not selected the applicable test layers for the current task, read
[the testing guide](testing.md) first. Read every applicable guide it selects.

## Keep the runtime harness small

The following complete test-file example proves one actual command round trip.
It uses a tiny root because the runtime, rather than a reusable leaf, implements
`tea.Model`. Project tests should pass their real root and controlled dependencies
through the same setup.

```go
package uitest

import (
    "bytes"
    "testing"
    "time"

    tea "charm.land/bubbletea/v2"
    "github.com/charmbracelet/colorprofile"
    "github.com/charmbracelet/x/exp/teatest/v2"
)

type completed struct{}
type runtimeRoot struct{ ready bool }

func (root runtimeRoot) Init() tea.Cmd {
    return func() tea.Msg { return completed{} }
}

func (root runtimeRoot) Update(message tea.Msg) (tea.Model, tea.Cmd) {
    switch message := message.(type) {
    case completed:
        root.ready = true
    case tea.KeyPressMsg:
        if message.String() == "q" {
            return root, tea.Quit
        }
    }
    return root, nil
}

func (root runtimeRoot) View() tea.View {
    if root.ready {
        return tea.NewView("ready")
    }
    return tea.NewView("loading")
}

func newRuntime(t testing.TB, root tea.Model) *teatest.TestModel {
    t.Helper()
    runtime := teatest.NewTestModel(t, root,
        teatest.WithInitialTermSize(40, 8),
        teatest.WithProgramOptions(
            tea.WithColorProfile(colorprofile.Ascii),
        ),
    )
    t.Cleanup(func() { runtime.GetProgram().Kill() })
    return runtime
}

func TestCommandReachesRoot(t *testing.T) {
    runtime := newRuntime(t, runtimeRoot{})
    teatest.WaitFor(t, runtime.Output(), func(output []byte) bool {
        return bytes.Contains(output, []byte("ready"))
    }, teatest.WithDuration(2*time.Second))

    runtime.Type("q")
    final := runtime.FinalModel(t, teatest.WithFinalTimeout(2*time.Second))
    root, ok := final.(runtimeRoot)
    if !ok || !root.ready {
        t.Fatalf("final model = %#v; want completed root", final)
    }
}
```

The runtime input and observation APIs serve different test needs:

| API | Purpose |
| --- | --- |
| `WithInitialTermSize` | Fix the starting terminal dimensions |
| `Send` | Inject a `tea.Msg` into the running program, including synthetic results, key events, or resize messages |
| `Type` | Send text as key presses |
| `Output` and `WaitFor` | Observe current or intermediate output until an explicit condition holds |
| `FinalModel` | Inspect the model after the program finishes |
| `FinalOutput` | Read remaining output after the program finishes |
| `RequireEqualOutput` | Compare captured output with its golden expectation |

For example, with `runtime` returned by `teatest.NewTestModel` and a root that
handles resize messages:

```go
runtime.Send(tea.WindowSizeMsg{Width: 20, Height: 6})
```

Unlike calling `Update` directly, `Send` crosses the real runtime message boundary.
Assert the resulting layout through output or final model state. For an injected
service result, send the application's actual typed message with the matching
owner/request identity. Sending a result tests routing and handling; it does not
prove the production service command produced that result. These input and
observation APIs are defined in the
[teatest v2 source](https://github.com/charmbracelet/x/blob/main/exp/teatest/v2/teatest.go).

ASCII color profile removes color variability for this runtime-flow test. A
color presentation test should pin its intended profile and theme instead.
`WaitFor` consumes output while observing it, so do not expect a later
`FinalOutput` read to contain the already-read prefix. Use separate tests or an
explicit capture when asserting the full stream.

Keep the experimental dependency pinned and centralize only stable setup such
as size, profile, and cleanup. The
[teatest v2 source](https://github.com/charmbracelet/x/blob/main/exp/teatest/v2/teatest.go)
defines these observation APIs; Bubble Tea defines the
[program options](https://github.com/charmbracelet/bubbletea/blob/main/options.go).
Do not grow this setup into another event framework.

