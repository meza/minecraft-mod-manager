# internal/perf

This package provides lightweight performance instrumentation built on `runtime/trace`.

It is used to:

- create trace regions around important operations (network calls, config I/O, downloads)
- keep an in-memory log of marks and measurements for tests and debugging

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

