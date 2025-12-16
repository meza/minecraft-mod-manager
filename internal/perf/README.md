# internal/perf

This package provides lightweight performance instrumentation built on `runtime/trace`.

It is used to:

- create trace regions around important operations (network calls, config I/O, downloads)
- keep an in-memory log of marks and measurements for tests and debugging

## Process lifecycle instrumentation

The app entrypoint (`main.go`) records top-level lifecycle regions so we can correlate end-to-end runtime with telemetry and subsystem-level perf regions.

Lifecycle region names are namespaced under `app.lifecycle.*`:

- `app.lifecycle.startup`: process start -> ready to begin command execution (wires telemetry + signal handlers)
- `app.lifecycle.execute`: command execution (CLI args or the interactive TUI session)
- `app.lifecycle.shutdown`: graceful or signal-triggered shutdown (flushes telemetry)

Each region creates:

- a start mark with the region name
- an end mark with `-end` appended
- a measurement with `-duration` appended

## Exporting perf logs

When you run the CLI with `--perf`, MMM writes `mmm-perf.json` when the process shuts down (gracefully or via Ctrl+C / SIGTERM).

By default, MMM writes the file next to the resolved `--config` path so it stays adjacent to the `modlist.json` you were working with.
Override the destination with `--perf-out-dir`. Relative output directories are resolved relative to the config directory.

The exported JSON includes marks and measurements. Any absolute filesystem paths stored in `PerformanceDetails` are normalized to be relative to the config directory so you can share the file without leaking machine-specific prefixes.

The JSON schema is intentionally simple:

- Marks include `timestamp`.
- Measurements include `start_timestamp` and `duration_ns`.

## Taxonomy (marker naming)

Perf markers are most useful when you can group them by "what part of the system is this?" and "what user action caused it?".
To keep that possible as the codebase grows, use dot-separated, lower-case, namespaced marker names.

## Performance strategy (ethos)

We instrument for diagnosis, not vanity metrics. The goal is that when someone says "feature X is slow", we can answer:

- Which part of the user flow dominates wall time?
- Is the time spent in user "thinking" vs system work?
- Which subsystem (platform orchestration, provider API, HTTP, filesystem) is the culprit?
- Which public entrypoint or retry loop is responsible?

This is why we favor layered instrumentation: broad, stable regions at the top and increasingly specific regions as you go down the stack.

### Depth and breadth (what we measure)

We intentionally cover the whole path from process start to user-visible completion:

- Process lifecycle: `app.lifecycle.*` brackets startup -> execution -> shutdown so all other regions can be correlated end-to-end.
- Commands and stages: `app.command.*` and `app.command.<cmd>.stage.*` break a command into user-meaningful phases (prepare, resolve, download, persist).
- Orchestration layers: `platform.*` measures stable public entrypoints like `platform.FetchMod` that coordinate provider calls and selection logic.
- Providers and transport:
  - Provider API calls: `api.<provider>.*` (project lookup, version listing, fingerprint match, etc).
  - Shared HTTP behavior: `net.http.*` (request, attempt, rate limit wait).
- Local I/O: `io.config.*`, `io.config.lock.*`, `io.download.*` (and `io.fs.*` when the filesystem is the bottleneck).
- Interactive sessions:
  - User actions and state transitions: `tui.<cmd>.state.*` and `tui.<cmd>.action.*`.
  - Thinking time: `tui.<cmd>.wait.<state>` so we can separate user dwell time from system latency.

### When to add markers

Add instrumentation when it helps a future collaborator make a confident performance call:

- A new public entrypoint becomes a dependency boundary (example: a new `internal/<module>` API used by commands).
- A user-visible step can be slow or flaky (network, disk, parsing, rendering, downloads).
- A loop/retry can amplify latency (HTTP retries, fallback version iteration, pagination).
- A feature has interactive branching paths (recoverable errors, "try again", choose platform, re-enter ID).

Avoid adding markers for tiny helpers that cannot meaningfully dominate time; prefer instrumenting the boundary that calls them.

### How to decide granularity

Use a "zoom ladder":

1. Add a region for the whole user intent (command or TUI session).
2. Add regions for the major phases that a user would recognize.
3. Add regions for stable module entrypoints and retry loops.
4. Only then add deeper regions for specific operations that are plausible bottlenecks.

This keeps the data queryable and avoids drowning in noise.

### Details (metadata) guidelines

Use `PerformanceDetails` for structured context, but keep it safe and low-cardinality:

- Prefer booleans and short enums: `success`, `provider`, `status`, `attempt`, `action`, `state`.
- Do not include secrets or user content.
- Avoid high-cardinality values in details (or normalize them), especially in interactive flows.
- If you need to correlate retries, include an `attempt` number and an `error_type` (using `fmt.Sprintf("%T", err)`).

### Non-blocking and testability

Instrumentation must never change command semantics or output.
If you touch instrumentation, add or update tests using the in-memory perf log:

- `perf.ClearPerformanceLog()` at test start
- assert the relevant marks/regions exist and include expected details

### Rules of thumb

- Use stable nouns, not transient implementation details (prefer `api.modrinth.project.get` over `modrinth-client-call`).
- Keep names short but specific; avoid encoding IDs in the marker name (use `PerformanceDetails` instead).
- Prefer one region per "unit of work" you would want to optimize or compare (a command, a platform call, a file write).
- Put retries/attempts in a child region (`...attempt`) so you can see both the total cost and the per-attempt cost.

### Namespaces

- `app.lifecycle.*`: process-level brackets (startup, execute, shutdown).
- `app.command.*`: a top-level unit of work the user asked for (cobra subcommand or the interactive session).
  - Examples: `app.command.add`, `app.command.init`, `app.command.list`, `app.command.version`, `app.command.tui`
- `platform.*`: orchestration over upstream providers (a stable place to measure public entrypoints like `platform.FetchMod`).
  - Examples: `platform.fetch_mod`
- `tui.*`: interactive UI work that is not a direct command execution.
  - Examples: `tui.render`, `tui.model.update`, `tui.prompt`
  - For user "thinking time", prefer `tui.<command>.wait.<state>` (for example `tui.add.wait.mod_not_found_confirm`).
- `api.<provider>.*`: outbound API calls grouped by platform provider.
  - Examples: `api.modrinth.project.get`, `api.modrinth.version.search`, `api.curseforge.project.get`, `api.curseforge.fingerprints.get`
- `net.http.*`: generic HTTP client behavior that is not provider-specific.
  - Examples: `net.http.request`, `net.http.request.attempt`, `net.http.ratelimit.wait`
- `io.fs.*`: filesystem reads/writes that might dominate local performance.
  - Examples: `io.fs.read`, `io.fs.write`, `io.fs.mkdir`, `io.fs.lock`
- `io.config.*`: config/lock specific file operations (when you want config to stand out from other filesystem I/O).
  - Examples: `io.config.read`, `io.config.write`, `io.config.init`, `io.config.lock.read`, `io.config.lock.write`
- `io.download.*`: downloads and other large byte transfers.
  - Examples: `io.download.file`

### Recommended details

Attach structured context via `PerformanceDetails` instead of encoding it in the name.
Common keys:

- `success` (bool) when a region can fail
- `provider` (string) for platform calls when the namespace alone is not enough
- `status` (int) for HTTP status codes
- `attempt` (int) for retry loops
- `bytes` (int64) for downloads/reads when available

### Current codebase note

Perf markers in the Go codebase should follow this taxonomy. If you find non-namespaced names, treat them as legacy and migrate them when you are already touching the surrounding code.

## Public API

- `StartRegion(name string) *PerformanceRegion`
- `StartRegionWithDetails(name string, details *PerformanceDetails) *PerformanceRegion`
- `(*PerformanceRegion).End()` / `EndWithDetails(...)`
- `Mark(name string, details *PerformanceDetails) *Entry`
- `Measure(name, fromMarker, toMarker string, details *PerformanceDetails)`
- `GetPerformanceLog() PerformanceLog`
- `GetAllMeasurements() PerformanceLog`
- `ClearPerformanceLog()`

`PerformanceDetails` is a map so callers can attach structured metadata (project ID, URL, etc).

## Tests

Run from the repo root:

```bash
make test
```
