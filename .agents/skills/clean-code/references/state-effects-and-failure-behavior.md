# State effects and failure behavior

Runtime behavior is legible and safe: what mutates and who owns it, what must happen in what order, what enters from outside, and what happens on every failure path.

Use the descriptions below to decide which individual quality references are relevant. Open only the references that can materially affect the current decision, code, or failure.

- [Side effects are visible](state-effects-and-failure-behavior/side-effects-are-visible.md) - A reader can tell whether a function mutates state, performs I/O, logs, throws, retries, or causes other external effects.
- [Hidden state is avoided](state-effects-and-failure-behavior/hidden-state-is-avoided.md) - A unit does not rely on ambient mutable state, invisible caches, implicit globals, or surprising external mutation.
- [Mutation has a clear owner](state-effects-and-failure-behavior/mutation-has-a-clear-owner.md) - When state changes, the owner, lifecycle, and allowed transitions are obvious.
- [Untrusted or invalid input is stopped at the boundary](state-effects-and-failure-behavior/untrusted-or-invalid-input-is-stopped-at-the-boundary.md) - External input is validated, normalized, sanitized, and rejected early enough that invalid or unsafe state does not spread through the system.
- [Error semantics are intentional and consistent](state-effects-and-failure-behavior/error-semantics-are-intentional-and-consistent.md) - Failures are surfaced in deliberate, diagnosable, and system-consistent ways, without silent semantic drift, swallowed exceptions, or arbitrary local error styles.
- [Temporal coupling is explicit](state-effects-and-failure-behavior/temporal-coupling-is-explicit.md) - If operations must happen in a particular order, that requirement is encoded or enforced rather than socially remembered.
- [Nondeterminism is controlled](state-effects-and-failure-behavior/nondeterminism-is-controlled.md) - Time, randomness, concurrency, and network variability are explicit and testable rather than leaking unpredictably into logic.
- [Resource handling is complete](state-effects-and-failure-behavior/resource-handling-is-complete.md) - Files, sockets, locks, transactions, and other resources are acquired and released correctly on both success and failure paths.
- [Correctness under concurrency, transactions, and retries is explicit](state-effects-and-failure-behavior/correctness-under-concurrency-transactions-and-retries-is-explicit.md) - The code makes ordering, atomicity, and safe re-entry understandable so concurrency, transaction handling, and repeated execution do not create hidden correctness risks.
- [The system is resilient to partial failure](state-effects-and-failure-behavior/the-system-is-resilient-to-partial-failure.md) - Dependency failure, timeout, bad input, and degraded subsystems do not cause disproportionate collapse.
- [Recovery paths are intentional](state-effects-and-failure-behavior/recovery-paths-are-intentional.md) - Restart, retry, replay, rollback, and repair stories exist where the domain requires them.

