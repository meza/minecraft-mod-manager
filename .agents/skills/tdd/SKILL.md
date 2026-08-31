---
name: tdd
description: Test-driven development and automated test quality. Use when writing or reviewing automated tests, including regression, component, API, database, integration, and end-to-end tests, or when the user asks for test-first development or mentions "red-green-refactor".
---

# Test-Driven Development

**Tautological tests considered harmful.**

TDD is the red → green loop. This skill is the reference that makes that loop produce tests worth keeping: what a good test is, where tests go, the anti-patterns, and the rules of the loop. Every section applies on every cycle — consult them before and during the loop, not after.

When exploring the codebase, read `CONTEXT.md` (if it exists) so test names and interface vocabulary match the project's domain language, and respect ADRs in the area you're touching.

## Decide whether TDD applies

Classify the change before starting a red-green cycle. TDD is required when application or library
behavior, a public contract, or executable logic changes through a stable test seam. Do not invent a
seam merely because a file changed or a task has an acceptance criterion.

Changes without a durable behavior contract do not require new automated tests. Common examples are
Markdown and comment edits, repository metadata, GitHub Actions workflow edits, and one-off mechanical
removals or renames. Validate those changes with the tools that match the work: documentation checks,
workflow syntax or schema validation, targeted repository searches, lint, typecheck, or build checks.
Existing regression suites may still be run when they provide proportionate confidence.

For a removal request such as "remove every reference to `X`", use a targeted search as completion
evidence. Do not turn that one-off search into a permanent test unless the user explicitly requires an
enduring policy and the repository has an established lint or static-analysis mechanism for it.

## What a good test is

Tests verify behavior through public interfaces, not implementation details. Code can change entirely; tests shouldn't. A good test reads like a specification — "user can checkout with valid cart" tells you exactly what capability exists — and survives refactors because it doesn't care about internal structure.

Carry that specification through suite and test names, helper names, assertions, and structure, not
through narration comments. Comments shown in examples explain this guide; they do not authorise
comments in repository code. The repository comment policy still governs every comment, including
tests, unless this skill declares an explicit exception.

See [tests.md](tests.md) for examples and [mocking.md](mocking.md) for mocking guidelines.

## Control time-dependent Vitest tests

When a Vitest test's setup or expected behaviour is relative to the current date or time, call
`vi.useFakeTimers()` and `vi.setSystemTime(...)` before creating that data or exercising the subject.
Derive dates and timestamps such as today, expired, future, age, elapsed time, and relative windows
from the frozen clock. Do not copy real calendar or timestamp strings into fixtures or expectations
as a substitute for controlling time. Restore real timers in guaranteed cleanup so the mocked clock
cannot leak into another test.

A date or timestamp literal remains appropriate when that exact value is an independent domain input
or expected contract and the behaviour does not interpret it relative to the current time. Playwright
tests use Playwright's clock API instead of Vitest's timer API.

## Seams — where tests go

A **seam** is the public boundary you test at: the interface where you observe behavior without reaching inside. Tests live at seams, never against internals.

**Test only at pre-agreed seams.** Before writing any test, write down the seams under test and confirm them with the user. No test is written at an unconfirmed seam. You can't test everything — agreeing the seams up front is how testing effort lands on the critical paths and complex logic instead of every edge case.

Ask: "What's the public interface, and which seams should we test?"

## Anti-patterns

- **Implementation-coupled** — mocks internal collaborators, tests private methods, or verifies through a side channel (querying the database instead of using the interface). The tell: the test breaks when you refactor but behavior hasn't changed.
- **Tautological** — the assertion recomputes the expected value the way the code does (`expect(add(a, b)).toBe(a + b)`, a snapshot derived by hand the same way, a constant asserted equal to itself), so it passes by construction and can never disagree with the code. Expected values must come from an independent source of truth — a known-good literal, a worked example, the spec.
- **Horizontal slicing** — writing all tests first, then all implementation. Bulk tests verify _imagined_ behavior: you test the _shape_ of things rather than user-facing behavior, the tests go insensitive to real changes, and you commit to test structure before understanding the implementation. Work in **vertical slices** instead — one test → one implementation → repeat, each test a **tracer bullet** that responds to what the last cycle taught you.

## Rules of the loop

- **Red before green.** Write the failing test first, then only enough code to pass it. Don't anticipate future tests or add speculative features.
- **One slice at a time.** One seam, one test, one minimal implementation per cycle.
- **Refactoring is not part of the loop.** It belongs to the review stage (see the `code-review-rules` skill), not the red → green implementation cycle.
