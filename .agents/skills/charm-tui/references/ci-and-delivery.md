# CI and delivery checks

Read this guide when establishing, changing, or reviewing TUI delivery checks.
The shared architecture and scope rules in [SKILL.md](../SKILL.md) apply.

## Establish a useful baseline

Use the repository's existing commands and tool versions. The table describes
the checks to cover; it does not require installing a second formatter, linter,
or task runner when the project already supplies one.

For a reusable Charm component repository, use this recommended CI baseline.
Local checks can target the change; CI retains the stated policy across the
library. Reusing project tooling preserves these checks rather than removing
their blocking status.

| Check | What it establishes | Recommended policy |
| --- | --- | --- |
| Project formatter (`gofumpt` or `gofmt`) | Consistent Go source | Blocking |
| `go vet ./...` and the configured linter, such as `golangci-lint` | Static mistakes and project conventions | Blocking |
| `go test ./...` | State, rendering, routing, and dependency contracts | Blocking |
| `go test -race ./...` | Exercised concurrent access, especially commands mutating UI state | Blocking CI job on a supported race platform |
| Golden comparison | Expected presentation under fixed inputs | Blocking; deliberate, inspected updates only |
| Small teatest suite | Actual runtime command and event delivery | Blocking; keep the suite small |
| Go module/tidy hygiene check | Dependencies match the packages that import them | Blocking for libraries, using the project's established check |
| Builds for supported operating systems | Platform-specific code compiles | Important for public packages; cover declared systems and architectures |
| Bubblebook catalogue and VHS recordings | Human interaction review and documentation | Generate or validate separately from correctness tests |
| Benchmarks and profiles of hot renderers | Performance regressions in measured operations | Threshold or report for known hot paths |

Use Bubblebook inspection for meaningful component states and VHS when a
recording serves documentation or review. Neither replaces the blocking tests.

Cross-compilation does not run the target system's tests. Race detection checks
only paths the tests exercise and needs a supported toolchain. Record those
limits instead of treating a local successful build as portability proof.
