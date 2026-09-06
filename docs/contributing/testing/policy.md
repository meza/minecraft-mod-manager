# Testing policy

This guide owns general test design and test-first development. Use the
[contribution verification policy](../../../CONTRIBUTING.md#verification) to select required commands
for the changed surfaces. Tests provide evidence for behavior; product requirements remain owned
by [product intent](../../intent.md) and the relevant [command guides](../../commands/README.md).

## Choose the test boundary

Use the narrowest stable interface that exposes the required behavior. Prefer real production
wiring and code paths. Control nondeterministic system boundaries such as time, randomness,
network, filesystem and OS signals, or force a rare error path with a focused fake. Do not replace
core behavior merely to make coverage pass.

| Contract | Guidance |
| --- | --- |
| Package behavior or an integration boundary | The affected package README and this policy |
| Component state, rendering, root composition or runtime delivery | [Component verification](components.md) |
| Operator capability through the real MMM process | [E2E architecture](README.md) and [BDD authoring](bdd.md) |
| Application layout, animation or styling | [Presentation testing](presentation.md) |
| HTTP responses across the process boundary | [HTTP fixtures](http-fixtures.md) |
| Native terminal input, waits, screen state or lifecycle | [Terminal harness](terminal-harness.md) |

Documentation, configuration and mechanical changes use their own verification routes; do not
invent production tests for work without an executable behavior contract.

## Behavior changes

Work through one behavior at a time:

1. Express the expectation in an automated test through the stable interface.
2. Run the test using the repository make target and verify it fails for the expected gap.
3. Implement the change that makes that expectation pass.
4. Refactor while keeping the tests green, then take the next behavior.

For user-facing behavior, capture the reviewed requirement as a third-person BDD scenario and
assert observable outcomes through the real MMM process. Every user-facing behavior change needs
at least one automated test that would fail if it regressed. If a behavior cannot be expressed in
a meaningful test, resolve the requirement or test boundary before implementing it.

Test success and relevant failure paths with independently stated expected outcomes. A test should
establish a caller or operator promise, not merely exercise lines or reproduce the implementation's
calculations. Coverage is required evidence, not a substitute for meaningful assertions.

## Reviewing and refactoring tests

Assess what the tests prove, whether a meaningful regression would fail, and whether they remain
useful after internal refactoring. Do not require evidence of authoring chronology in a test review.
For a behavior-preserving refactor, existing suitable tests can supply the contract; do not invent
a behavior change simply to create a new test.

## Terminal outcomes and presentation

Run shared BDD scenarios unchanged across execution profiles. Drivers execute actions; shared
product assertions verify capabilities; separately owned presentation checks reuse suitable
journeys. Use [E2E architecture](README.md) for those boundaries.

Compare durable records and relevant resolved decisions under the
[permanent transcript contract](../../intent.md#active-display-and-permanent-transcript). Do not
compare raw control bytes or input exchanges, or normalize away meaningful discrepancies.

Prefer stable i18n keys, interpolation arguments, exit status, filesystem effects and semantic
terminal state over rendered wording. Use full terminal snapshots when the reviewed requirement
depends on complete layout or styling, and update them only for intentional product changes.
Direct component checks complement rather than replace real terminal evidence. tui-test owns
process-terminal input, waits, screen state, lifecycle and snapshots; do not add project-owned
PTY or terminal emulation helpers.

## Commands and retained snapshots

Use repository make targets rather than direct Go test or build commands. The
[verification policy](../../../CONTRIBUTING.md#verification) owns gates and their applicability.
The [terminal harness prerequisites](terminal-harness.md#prerequisites) and
[HTTP fixture guide](http-fixtures.md) apply when running their respective E2E work.

`make coverage` runs the unified coverage tool, generates `coverage.html` and the function report
in `coverage.out`, and enforces 100% coverage after configured exclusions.
`make lint` and `make lint-fix` use the golangci-lint version pinned by the project.

After an intentional change to a retained non-E2E Go snapshot, update it with:

```sh
UPDATE_SNAPS=true make coverage
```

In PowerShell, set `UPDATE_SNAPS` to `true` in the command's environment before running
`make coverage`, then restore its previous value. Inspect the updated baseline. Native terminal
snapshots follow the [terminal harness workflow](terminal-harness.md), not this
non-E2E update command.
