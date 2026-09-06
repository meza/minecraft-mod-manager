# Terminal debugging

Read this guide when diagnosing a running TUI with logs or a debugger.
The shared architecture and scope rules in [SKILL.md](../SKILL.md) apply.

## Debug without corrupting the terminal

While the program owns the terminal, send debug output to a file. In a
function returning `error`, before calling `tea.NewProgram(...).Run()`:

```go
debugFile, err := tea.LogToFile("debug.log", "ui")
if err != nil {
    return err
}
defer debugFile.Close()
```

This redirects the default Go logger. Inside the root's update path, a temporary
`log.Printf("message=%T", message)` records message types without printing user
text. Include safe instance and request identifiers when diagnosing stale-result
routing. Avoid dumping arbitrary message values that may contain credentials,
clipboard contents, or private input. Remove temporary noisy instrumentation
after finding the cause. The
[Bubble Tea logging helper](https://github.com/charmbracelet/bubbletea/blob/main/logging.go)
owns file setup and supports a supplied logger through `LogToFileWith`.

Follow the log from another terminal with `tail -f debug.log` on a POSIX shell,
or `Get-Content -Wait debug.log` in PowerShell. For breakpoints, start Delve in
the application's main-package directory:

```console
dlv debug --headless --api-version=2 --listen=127.0.0.1:43000 .
```

Connect from a second terminal:

```console
dlv connect 127.0.0.1:43000
```

Then set the desired breakpoint and continue from that debugger session. The
application keeps its terminal while the debugger uses the other session.
[Bubble Tea's debugging instructions](https://github.com/charmbracelet/bubbletea#debugging)
describe this arrangement. Live reload can shorten manual iteration when the
project already uses it; it does not alter component lifecycle contracts.

Document the project's actual headless Delve invocation and attachment steps in
its existing `CONTRIBUTING.md`, together with the logging setup and log-following
commands. Component authors need a durable contributor entry point for these
commands, so they can debug without printing into the controlled terminal.
