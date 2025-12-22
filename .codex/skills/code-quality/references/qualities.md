# Code Quality Dimensions

## Table of Contents

- [Readability](#readability)
- [Correctness](#correctness)
- [Reliability](#reliability)
- [Operability](#operability)
- [Efficiency](#efficiency)
- [Maintainability](#maintainability)
- [Safety](#safety)
- [Consistency](#consistency)

---

## Readability

Good code communicates its intent to human readers. A developer encountering the code for the first time can understand what it does and why.

### Clarity

#### Expressive Naming

Names are domain-specific and descriptive. A name communicates what something is or does without requiring the reader to trace through implementation. Where a generic term like "manager" or "handler" might appear, a domain-specific term that conveys actual meaning appears instead. Single-character names appear only in tightly scoped contexts like loop indices where their meaning is unambiguous.

#### Domain-Aligned Structure

Code organization reflects the problem domain. Concepts that belong together live together. Boundaries between domains are clear. The structure makes navigation intuitive because it mirrors how domain experts think about the problem.

#### Self-Documenting Code

Comments explain why, not what. The code itself, through names, types, and structure, communicates what it does. Comments exist for non-obvious constraints, historical context, or reasoning that cannot be expressed in code. Public APIs have docblocks that describe contracts.

### Simplicity

#### Minimal Solutions

The simplest approach that solves the problem is preferred. Clever solutions give way to clear ones. When multiple approaches exist, the one with fewer moving parts wins unless there is concrete evidence that complexity pays for itself.

#### Earned Abstractions

Abstractions emerge from demonstrated, repeated need. An abstraction exists because multiple concrete use-cases demanded it, not because someone imagined future use-cases might. Three similar implementations coexist comfortably until the pattern is clear; premature unification is more costly than temporary duplication.

#### Small Surfaces

Public interfaces expose only what consumers need. Every additional public function, method, or type is a commitment. What can remain private does remain private. Internal implementation details do not leak through the interface.

### Explicitness

#### Visible Dependencies

Dependencies are declared and injected, not discovered or assumed. Looking at a unit's construction tells you what it needs. No magic discovery, no ambient singletons, no implicit service locators.

#### Visible Configuration

Configuration comes from outside the code: environment, files, parameters. Not from hard-coded values buried in implementation. The code's behavior can be understood and modified without changing source.

#### Visible State

When state exists, its scope and lifecycle are clear. Global mutable state is absent or explicitly isolated. Data flows through parameters and return values. Side effects are visible in function signatures or naming conventions.

#### Visible Assumptions

When code depends on conditions it cannot verify, environmental constraints, upstream guarantees, or timing assumptions, those assumptions are stated. The reader knows what must be true for the code to work correctly.

### Interface Design

#### Consistent APIs

Public interfaces follow consistent patterns. Similar operations have similar signatures. Naming conventions are uniform. A developer who learns one part of the API can predict how other parts behave.

#### Versioned Contracts

APIs that have external consumers are versioned. Breaking changes are explicit and managed. Consumers can depend on stability within a version.

#### Backward Compatibility

Changes to public interfaces preserve compatibility with existing consumers unless breaking changes are explicitly communicated and justified. Migration paths exist when contracts must change.

---

## Correctness

Good code does what it claims to do, and demonstrates this through verification.

### Design for Testability

#### Single Responsibility

Each unit has one clear purpose. A function does one thing. A class represents one concept. A module owns one bounded context. When describing what a unit does, "and" rarely appears. Units with single responsibilities are straightforward to test.

#### Composition Over Inheritance

Shallow composition is preferred over deep inheritance hierarchies. When inheritance appears, the relationships are genuine "is-a" relationships. Deep hierarchies with overridden behavior throughout are difficult to test and reason about.

#### Injectable Dependencies

Dependencies can be provided at construction time, enabling tests to substitute controlled implementations where necessary. The code does not reach out to discover its dependencies through global state or service locators.

### Predictability

#### Deterministic Behavior

Given the same inputs in the same state, the code produces the same outputs. Randomness, time-dependence, and external state are isolated and controllable. Surprising variations in behavior do not occur.

#### No Hidden Nondeterminism

When behavior must vary due to randomness, time, or external factors, this is explicit in the interface. The sources of variation are visible and can be controlled for testing.

### Testing Philosophy

#### Tests Map to Behavior

Each significant behavior has at least one test that would fail if the behavior changed. Tests document what the code promises. The connection between behavior and test is traceable.

#### Tests Precede Implementation

For new or significant behavior, the test exists before the implementation. The test defines what "correct" means. The implementation satisfies the test. When test-first is impractical for small fixes, tests are added before the change is complete.

#### Tests Cover Failure Cases

Tests cover what happens when things go wrong: invalid input, missing data, failed dependencies, boundary conditions. The code's behavior under failure is as intentional and verified as its behavior under success.

### Testing Practice

#### Tests Exercise Real Code

Tests run the actual production code paths whenever possible. Test doubles appear only where necessary: to control nondeterminism (time, randomness, network, filesystem) or to force rare conditions (disk full, connection timeout). The code path a test exercises is the code path production uses.

#### Tests Are Deterministic

A test produces the same result every time it runs. No flakiness, no timing dependencies, no order dependencies. A failing test means the code is wrong, not that the test is unreliable. Flaky tests are isolated and tracked for remediation; they are never ignored or normalized.

#### Tests Are Fast

Tests run quickly enough to run frequently. Slow tests are separated and labeled. The fast test suite provides rapid feedback during development. Test organization supports parallel execution.

#### Tests Are Never Hidden

Tests are not changed or skipped to force a green build. When behavior legitimately changes, tests are updated with clear reasoning. Skipped tests without documented rationale and remediation plan do not persist.

---

## Reliability

Good code behaves predictably under normal conditions and degrades gracefully under stress.

### Error Handling

#### Explicit Error Paths

The possible failures are visible in the code's interface through types, return values, or documented exceptions. A caller knows what can go wrong without reading the implementation.

#### Loud Failures

Invalid states are detected and reported, not silently ignored. When something is wrong, the code says so. Silent failures that corrupt data or mislead users do not occur.

#### Boundary Validation

Input from outside the system (users, APIs, files, network) is validated at entry. Once validated, the data flows through the system without repeated checking. Internal code trusts the invariants established at boundaries.

#### Designed Recovery

When failures occur, the response is intentional. Retries use backoff. Failures are isolated to prevent cascading. Degraded modes are explicit. The system fails gracefully rather than catastrophically.

### Resource Management

#### Acquire-Release Discipline

Resources that are acquired are released. Files are closed. Connections are returned to pools. Locks are released. Memory is freed when no longer needed. The code does not leak resources under any code path, including error paths.

#### Scoped Lifetimes

Resource lifetimes are tied to lexical scope or explicit ownership. Language constructs like try-with-resources, defer, using, or RAII patterns ensure cleanup happens automatically. Manual cleanup spread across the codebase is avoided.

#### Bounded Resource Usage

Resource consumption has limits. Connection pools have maximum sizes. Buffers have bounds. Queues have capacity limits. Unbounded growth that could exhaust system resources does not occur.

### Resilience Patterns

#### Timeouts on External Calls

Every call to an external system (network services, databases, file systems) has a timeout. The code does not wait indefinitely for responses that may never come. Timeout values are appropriate to the operation.

#### Circuit Breakers

Repeated failures to external dependencies trigger circuit breakers that prevent continued attempts. The system degrades rather than waiting indefinitely or overwhelming failing services.

#### Bulkheads

Failure in one part of the system does not cascade to unrelated parts. Resource pools are isolated. One misbehaving component cannot exhaust resources needed by others.

#### Graceful Degradation

When components fail, the system continues operating with reduced functionality rather than failing entirely. Fallback behaviors are explicit and tested.

#### Idempotent Operations

Operations that may be retried produce the same result when executed multiple times. Network failures, timeouts, and retries do not cause duplicate effects. Where true idempotency is impossible, duplicate detection or compensation mechanisms exist.

### Concurrency

#### Thread-Safe Patterns

Concurrent code uses established idioms for safety. Data races cannot occur. Ownership of mutable state is clear and enforced. Race detectors are used where available.

#### Documented Concurrency Constraints

When code assumes sequential access, single-threaded execution, or specific timing, those assumptions are stated. The reader knows what concurrency guarantees the code requires.

### Data Integrity

#### Reversible Migrations

Schema changes can be undone. Migrations are tested on representative data before production. The path back exists. Large migrations are staged and monitored.

#### Protected Invariants

Data invariants are enforced at the appropriate layer: schema constraints, validation logic, or application rules. Invariants do not rely solely on application code that might be bypassed.

---

## Operability

Good code reveals its runtime behavior and operates well in production.

### Observability

#### Structured Telemetry

Logs are structured and machine-parseable. Metrics cover the important dimensions: request rates, error rates, latencies at meaningful percentiles (p50, p95, p99). Naming conventions are consistent across the system.

#### Traceable Requests

Requests carry correlation IDs through the system. A request can be traced from entry to exit across service boundaries. When something goes wrong, the path is visible in logs and traces.

#### Informative Errors

Error messages contain enough context to understand what happened and why. They guide diagnosis without exposing sensitive information. Stack traces and correlation IDs appear where appropriate.

### Startup and Shutdown

#### Configuration Validation at Startup

Configuration is validated when the application starts. Missing values, invalid formats, and inconsistent settings cause immediate, clear failures. The application does not start in a broken state only to fail later at runtime.

#### Graceful Shutdown

The application handles termination signals properly. In-flight requests complete or are cleanly aborted. Connections drain. Resources are released. The shutdown process is orderly, not abrupt.

### Health Exposure

#### Health Checks

Services expose their health status through standard endpoints. Liveness checks indicate the process is running. Readiness checks indicate the service can handle requests. Health checks are lightweight and accurate.

#### Dependency Health

Health checks reflect the status of critical dependencies. A service reports unhealthy when it cannot fulfill its purpose due to downstream failures.

---

## Efficiency

Good code uses resources appropriately without premature optimization.

### Performance Awareness

#### No Needless Waste

Code does not perform obviously inefficient operations when equally clear alternatives exist. O(n squared) algorithms are not used when O(n) solutions are equally readable. Unnecessary allocations, copies, and iterations are avoided.

#### Appropriate Data Structures

Data structures match access patterns. Lookups use maps or sets, not linear scans through lists. Sorted data uses appropriate search. The choice of structure reflects how the data is used.

#### Lazy Computation

Expensive operations are deferred until needed. Large data sets are processed incrementally or streamed rather than loaded entirely into memory when possible.

### Optimization Discipline

#### Measured, Not Assumed

Performance improvements are based on measurement, not intuition. Profiling identifies actual bottlenecks. Optimizations target measured problems.

#### Preserved Clarity

Optimizations do not sacrifice clarity without justification. When performance requires complex code, the complexity is isolated and documented. The simple path remains available for understanding.

---

## Maintainability

Good code is safe and easy to change over time.

### Structural Qualities

#### Small Units

Functions are short. Classes are focused. Files are navigable. Nothing is so large that understanding it requires heroic effort. Small units are easier to understand, test, and replace.

#### High Cohesion

Elements within a module belong together. A module's contents are related and work toward a common purpose. Unrelated functionality lives elsewhere. When you need to change something, the relevant code is in one place.

#### Loose Coupling

Modules depend on each other through narrow, stable interfaces. Changes to one module's internals do not ripple through the codebase. Dependencies point in one direction. Circular dependencies do not exist.

#### Localized Changes

Changing one behavior does not require changes scattered across the codebase. Related code lives together. Coupling between distant parts is minimal. A change's blast radius is predictable.

#### Incremental Evolution

The code can be changed in small steps. Large changes decompose into smaller, independently viable changes. The path from current state to desired state has stable intermediate points. Each step is reviewable and reversible.

### Debt Management

#### Visible Debt

When shortcuts exist, they are marked. Technical debt is tracked explicitly, not hidden in the codebase. TODOs convert to tracked items. The cost of past decisions is visible so future decisions can account for it.

#### Scoped Refactoring

Refactoring stays within the current task's scope. Broad refactors do not mix with feature work. When larger refactoring is needed, it is proposed separately with clear boundaries and rollback strategy.

---

## Safety

Good code protects users, data, and systems from harm.

### Security

#### No Embedded Secrets

Credentials, keys, tokens, and passwords do not appear in source code. Secrets enter through secure runtime channels. The code can be made public without exposing access. If secrets are discovered in code, they are revoked and rotated immediately.

#### Input Sanitization

External input is never trusted. Values are validated and sanitized before use. Injection attacks (SQL, command, script) cannot succeed because untrusted data never reaches dangerous contexts unsanitized.

#### Minimal Privilege

Code requests only the access it needs. Permissions default to denied. Capabilities are scoped narrowly. Access controls are enforced, not advisory.

### Privacy

#### Data Minimization

Personal data collection is limited to what is necessary. Data is not retained longer than needed. The code does not enable fishing expeditions through user data.

#### Controlled Access

Access to personal data is logged and auditable. The code respects user consent and privacy preferences. Legal and policy requirements are reflected in the implementation.

### Inclusivity

#### Accessible Interfaces

User-facing code meets accessibility standards appropriate to the project. Users with disabilities can use the software. Accessibility is designed in from the start, not retrofitted.

#### Inclusive Language

APIs, logs, documentation, and code use inclusive terminology. Language choices are thoughtful. Harmful or exclusionary terms are avoided.

---

## Consistency

Good code respects the project's established patterns and standards.

### Project Standards

#### Style Conformance

Code follows the project's style guide and conventions. Formatting is consistent. Naming patterns match existing code. A reader moving through the codebase does not encounter jarring style shifts.

#### Idiomatic Patterns

Solutions use patterns established in the project. When the project has a way of doing something, new code follows that way unless there is explicit reason to diverge. Divergence is documented.

### Automated Enforcement

#### Static Analysis

Linters, formatters, and type-checkers run as part of development and CI. Style and correctness rules are enforced automatically, not through manual review. Deviations are documented with rationale.

#### Security Scanning

Dependency vulnerability checks and static security analysis run regularly. Findings are addressed or tracked with mitigation plans.
