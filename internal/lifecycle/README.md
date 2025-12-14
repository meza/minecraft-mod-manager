# Lifecycle Manager

`internal/lifecycle` centralizes Ctrl+C / SIGTERM handling so subsystems can flush state without duplicating boilerplate. It guarantees that handlers run once (in reverse registration order) and maps signals to conventional exit codes.

## Why it exists

- Commands and TUIs often need to flush telemetry, close network clients, or revert temp files.
- Multiple modules may need shutdown hooks at the same time.
- Signal handling must be consistent and safe even when one handler panics.

## Registering handlers

```go
import (
    "context"
    "os"

    "github.com/meza/minecraft-mod-manager/internal/lifecycle"
)

func main() {
    handlerID := lifecycle.Register(func(os.Signal) {
        // perform cleanup
    })
    defer lifecycle.Unregister(handlerID)

    // run CLI/TUI logic; normal returns should still defer cleanup
}
```

- `Register` ignores `nil` callbacks, starts the SIGINT/SIGTERM listener on first use, and returns a `HandlerID`.
- `Unregister` removes the handler when it’s no longer needed (useful if the owning subsystem shuts down early).

Handlers fire only when the OS delivers SIGINT or SIGTERM. Normal command exits should still rely on defers or command-level cleanup.

## Testing behaviour

Public APIs already account for common failure cases:

- Handlers run in reverse order so the most recently registered subsystem can shut down first.
- Panics inside a handler are swallowed so the remaining callbacks still run.
- Unregister removes handlers so temporary subsystems don’t leak callbacks.

Internally there is a `reset` helper and injectable channel factory for unit tests. Consumers do not need access to those helpers.

## Related docs

- `internal/telemetry/README.md` shows how telemetry registers its shutdown hook.
