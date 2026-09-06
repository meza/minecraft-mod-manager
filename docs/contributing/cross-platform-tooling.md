# Cross-platform tooling

Repository tooling must behave consistently on supported Windows, Linux, and macOS environments.
This applies to package scripts, repository scripts, Git hooks, setup and maintenance commands, checks,
test harnesses, and reusable workflow logic.

## Use portable execution boundaries

Prefer Go, existing project tools and suitable cross-platform dependencies for shared tooling.
Introduce another runtime only when the requested capability requires it and the dependency is
justified. Invoke executables
with an argument array through a process API. Do not construct shell command strings or depend on
pipes, redirects, command substitution, inline environment assignment, glob expansion, or quoting
performed by a particular shell.

Keep reusable tooling logic independent of Bash, PowerShell, POSIX or GNU utilities. Documented
Make entry points may select the native platform shell for orchestration. Workflow syntax may use
the declared runner's shell for orchestration confined to that runner; reusable logic and the
implementation of a check belong in cross-platform code.

## Handle files and processes through platform APIs

Construct filesystem paths with the runtime's path APIs. Do not assume `/` or `\` separators,
drive-letter or root syntax, case sensitivity, executable suffixes, newline sequences, temporary
directory locations, or symbolic-link availability.

Normalize paths to POSIX form only at a boundary whose external contract requires that form, such as
a Git path, URL, or package export key. Keep native filesystem paths native everywhere else.

Resolve temporary locations, environment variables, executable names, and subprocess behaviour
through platform-aware APIs. Treat different line endings, spaces and non-ASCII characters in paths,
and unavailable symbolic links as supported operating conditions.

## Keep platform-specific work isolated

An inherently platform-specific operation must name its platform in its task or script, document why
that restriction is intrinsic, and remain isolated from repository-wide contributor workflows. A
platform-specific implementation is not an acceptable substitute for a shared tooling entry point.

## Review portability

Review every tooling change against the supported Windows, Linux, and macOS environments. Trace its
filesystem, process, environment, and shell boundaries and block assumptions that are not portable.
Portability is an implementation and review requirement, not a testing requirement.

Tooling tests are optional. Add them when complex logic would benefit from regression protection,
not solely to demonstrate portability. They may run on the repository's configured CI environment
and do not require an operating-system matrix.

Use the [contribution verification policy](../../CONTRIBUTING.md#verification) for mixed changes.
The optional tooling tests above do not waive tests for changes to production behavior.
