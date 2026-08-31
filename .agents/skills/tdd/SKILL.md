---
name: tdd
description: Test-driven development and automated test quality. Use when implementing behavior test-first, writing or reviewing automated tests, or when the user mentions TDD, test-first development, regression tests, or red-green-refactor.
---

# Test-driven development

Use the mode that matches the task. Do not impose implementation chronology on a test review.

## Establish the test seam

Before writing or assessing tests, inspect the relevant architecture documentation, public interfaces, existing tests, and nearby repository guidance. Identify the narrowest stable interface through which callers or users observe the behavior.

Use an established seam when the evidence identifies one. Ask the user only when materially different seams remain plausible and the choice would change the behavior contract or scope.

Read [tests.md](tests.md) when designing or reviewing test cases. Read [mocking.md](mocking.md) when the work needs control over a system boundary.

## Choose a mode

### Implementation mode

Use implementation mode when changing application or library behavior through a stable test seam.

Work in vertical slices:

1. Red: add one test that expresses an observable behavior and run it to prove the expected failure.
2. Green: implement only enough behavior to make that test pass.
3. Refactor: improve the code and test design while keeping the suite green.
4. Repeat for the next behavior.

Do not write all tests before all implementation. Each cycle should use what the previous cycle revealed.

User-visible behavior requires the snapshot coverage defined by `CONTRIBUTING.md`. Run repository `make` targets rather than invoking Go test or build commands directly.

### Test-review mode

Use test-review mode when the request is to assess existing or proposed tests.

Do not require proof that the reviewed tests were authored before the implementation. Assess whether they:

- exercise observable behavior through stable interfaces;
- would fail for a meaningful regression;
- derive expected values independently from the implementation;
- cover relevant success and failure behavior;
- control nondeterministic boundaries;
- follow repository snapshot and coverage requirements; and
- remain useful after internal refactoring.

Report concrete gaps and their consequences. Do not modify code unless the user has authorized changes.

## Decide whether automated tests apply

Do not invent a test seam for work without a durable behavior contract. Markdown, comments, repository metadata, workflow configuration, and one-off mechanical removals normally use documentation checks, schema validation, targeted searches, lint, or build checks instead.

For requests such as removing every reference to a name, use a targeted search as completion evidence. Add a permanent test only when the absence is an enduring policy with an established enforcement mechanism.

## Test qualities

- Test behavior that users or callers can observe through a stable interface.
- Name tests in domain language without ticket IDs or delivery metadata.
- Use an independent source of truth for expected values.
- Keep a test focused on one behavior, while allowing the assertions needed to prove it.
- Control time, randomness, network, filesystem, and other nondeterministic boundaries explicitly.
- Prefer real production wiring. Replace only boundaries that must be controlled.

## Self-verification

Before completing the work, verify that:

- the chosen mode matches the task;
- the seam follows repository architecture and existing public interfaces;
- implementation work includes evidence of red, green, and refactor cycles;
- test review does not claim or require unobservable authoring chronology;
- tests are sensitive to meaningful regressions rather than implementation details; and
- validation uses the repository checks required by `CONTRIBUTING.md`.
