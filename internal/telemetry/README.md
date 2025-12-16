# internal/telemetry

Minecraft Mod Manager ships anonymous usage metrics to PostHog (https://posthog.com) so we can understand which commands people reach for and where errors cluster.

Telemetry is best-effort: failures never block a command. By default, the telemetry package uses a no-op logger, so telemetry failures are silent unless the package logger is explicitly wired for debugging.

## Quick start

This is the pattern used by `main.go`:

```go
telemetry.Init()
handlerID := lifecycle.Register(func(os.Signal) {
	telemetry.Shutdown(context.Background())
})
defer lifecycle.Unregister(handlerID)
defer telemetry.Shutdown(context.Background())
```

Call `Init` once when the process starts, emit events via `Capture`/`CaptureCommand`, and rely on `internal/lifecycle` to flush telemetry during Ctrl+C/SIGTERM. Keep a `defer telemetry.Shutdown(...)` for the graceful exit path.

## Runtime lifecycle

1. `telemetry.Init` gathers configuration, honours opt-out flags, and creates the PostHog client.
2. Commands call `telemetry.Capture` or `telemetry.CaptureCommand` whenever they have data to record. The same client is reused across the entire process so multi-command TUI sessions only initialise telemetry once.
3. `telemetry.Shutdown` flushes pending events and closes the client. The entry point registers this cleanup with `internal/lifecycle`, so it runs on normal exit as well as when Ctrl+C / SIGTERM fire, and future subsystems can attach their own shutdown hooks alongside telemetry.

Because the lifecycle is explicit, the TUI can keep telemetry active while users jump between screens and then defer a single `Shutdown` when the UI loop ends.

## Opt-out and overrides

- Set `MMM_DISABLE_TELEMETRY=1` (`1`, `true`, `yes`, `on`) to disable telemetry entirely.
- Provide `MACHINE_ID=<value>` to override the default hardware fingerprint (useful for reproducible tests or CI).

## Failure behaviour

Telemetry must never impact user flows:

- Missing API keys or opt-out variables short-circuit initialisation.
- `Capture` simply returns when telemetry is disabled.
- Errors from `Init`, `Capture`, or `Shutdown` only emit debug logs and are ignored otherwise.

Future command implementations should continue following this pattern: emit telemetry in the background, but always prioritise the CLI/TUI behaviour over analytics.
