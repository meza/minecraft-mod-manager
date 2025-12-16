# internal/lifecycle

This package is where we centralize Ctrl+C (SIGINT) and SIGTERM handling so subsystems can flush state without duplicating signal boilerplate.

It guarantees:

- handlers run at most once per process
- handlers run in reverse registration order (last registered shuts down first)
- a panic in one handler does not prevent others from running
- the process exits with a conventional exit code for the signal

## Quick start

Register cleanup hooks from wherever you own a resource that should be flushed on shutdown.

```go
handlerID := lifecycle.Register(func(sig os.Signal) {
	_ = sig
	// Perform cleanup.
})
defer lifecycle.Unregister(handlerID)
```

Handlers only run on signals. Normal command exits should still rely on `defer` or command-level cleanup.

## Public API

- `Register(handler Handler) HandlerID`
- `Unregister(id HandlerID)`

`Register` ignores `nil` handlers, starts the SIGINT/SIGTERM listener on first use, and returns a `HandlerID` you can use to unregister.

`Unregister` removes the handler when it is no longer needed (useful when a temporary subsystem shuts down early).

## Exit codes

Signals are mapped to conventional exit codes:

- SIGINT (Ctrl+C) -> 130
- SIGTERM -> 143

## Testing notes

This package has a test-only `reset` helper and injectable factories so unit tests can run without real OS signals.

Consumers should treat those helpers as private implementation details and only depend on the public API guarantees.

## Related docs

- `internal/telemetry/README.md` shows how telemetry shuts down on Ctrl+C/SIGTERM.
