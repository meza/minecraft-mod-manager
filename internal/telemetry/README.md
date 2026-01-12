# internal/telemetry

Minecraft Mod Manager ships anonymous usage metrics to PostHog (https://posthog.com) so we can understand which commands people reach for, where errors cluster, and where time is spent. Telemetry uses a stable machine identifier that is not PII so we can track long-term behavior without collecting personal data.

Telemetry is best-effort: failures never block a command. By default, the telemetry package uses a no-op logger, so telemetry failures are silent unless the package logger is explicitly wired for debugging.

## Quick start

This is the pattern used by `main.go`:

```go
telemetry.Init()
var shutdownOnce sync.Once
shutdown := func(sig os.Signal) {
	shutdownOnce.Do(func() {
		telemetry.Shutdown(context.Background())
	})
}

handlerID := lifecycle.Register(func(sig os.Signal) {
	shutdown(sig)
})
defer lifecycle.Unregister(handlerID)
defer shutdown(nil)
```

Call `Init` once when the process starts, record command outcomes via `RecordCommand`, and rely on `internal/lifecycle` to flush telemetry during Ctrl+C/SIGTERM. Keep a `defer telemetry.Shutdown(...)` for the graceful exit path.

## Perf correlation

`main.go` brackets the telemetry lifecycle inside `internal/perf` regions so perf marks can be correlated with telemetry activity:

- `app.lifecycle.startup` includes `telemetry.Init()`
- `app.lifecycle.shutdown` ends before the telemetry flush, so the perf export tree is complete when `telemetry.Shutdown(...)` builds the perf summary

Session telemetry includes a `performance` payload (perf_summary_v1 schema: app version, OS, execution mode, ordered commands, per-command timing + stage timing when available, modlist context when config is available, and request/download counts), plus top-level `total_time_ms` and `work_time_ms` (total runtime minus `interaction.*.wait.*` thinking time). The raw perf span tree is only written to `mmm-perf.json` when `--perf` is used.
Some interactive flows use `interactive.*.wait.*` spans for thinking time as well.

### performance schema (perf_summary_v1)

All durations are milliseconds.

Top-level fields:

- `schema_version` (int)
- `app_version` (string)
- `os` (object: `goos` string, `goarch` string)
- `execution_mode` (string: `interactive`, `unattended`, `non_tty`, or `unknown`)
- `commands` (array of command summaries, in execution order)
- `modlist` (object: `game_version` string, `loader` string, `mod_count` int; omitted if config cannot be read)
- `counts` (object: `http_requests` int, `downloads` int, `download_bytes` int64 in bytes when available)

Command summary fields:

- `name` (string)
- `success` (bool)
- `exit_code` (int)
- `execution_mode` (string)
- `error_category` (string, optional)
- `error` (string, optional)
- `extra` (object, optional)
- `arguments` (object, optional)
- `duration_ms` (int64, optional)
- `stage_durations_ms` (object map of stage name to duration, optional)

## Runtime lifecycle

1. `telemetry.Init` gathers configuration, honours opt-out flags, and creates the PostHog client.
2. Commands call `telemetry.RecordCommand` when they finish (success or failure). Telemetry is stored in-memory for the session.
3. `telemetry.Shutdown` emits a single session-level event, flushes pending events, and closes the client. The entry point registers this cleanup with `internal/lifecycle`, so it runs on normal exit as well as when Ctrl+C / SIGTERM fire, and future subsystems can attach their own shutdown hooks alongside telemetry.

The session-level event name is derived in this order:

1. The single recorded command name (when exactly one command was recorded).
2. The top-most `app.command.<name>` perf span (so aliases do not change the taxonomy).
3. The session name hint supplied by the entrypoint (`tui` for interactive sessions, otherwise the argv token).

Because the lifecycle is explicit, the TUI can keep telemetry active while users jump between screens and then defer a single `Shutdown` when the UI loop ends.

## Opt-out and overrides

- Set `MMM_DISABLE_TELEMETRY=1` (`1`, `true`, `yes`, `on`) to disable telemetry entirely.
- Provide `MACHINE_ID=<value>` to override the default hardware fingerprint (useful for reproducible tests or CI).

## Failure behaviour

Telemetry must never impact user flows:

- Missing API keys or opt-out variables short-circuit initialisation.
- `Capture` simply returns when telemetry is disabled.
- Errors from `Init`, `Capture`, or `Shutdown` only emit debug logs and are ignored otherwise.

Future command implementations should continue following this pattern: emit telemetry in the background, but always prioritise the terminal behavior over analytics.
