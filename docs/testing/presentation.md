# Presentation checks on product journeys

This guide owns how presentation requirements obtain evidence without putting
visual assertions into shared product scenarios or drivers. Read [BDD architecture](bdd.md)
for responsibilities and the [terminal harness](terminal-harness.md) for observation
APIs, lifecycle and prerequisites.

## Share execution, separate assertions

A journey performs product work; checks evaluate what happened. Product and
presentation checks can observe the same MMM process without becoming one assertion
or requiring a second copy of the journey.

The runner selects applicable presentation checks outside shared Gherkin. Attach a
check to a journey/profile that reaches its required state. Selection is optional
for an individual product-only run, but all required checks must execute and pass
before acceptance. Not applicable is not passed; omitting a required check leaves
verification incomplete.

Drivers perform actions and expose evidence through tui-test's native Go binding. They
contain no spinner, styling, layout or business-outcome assertions. Presentation
checks consume observations; they do not independently send competing input or own
the process. The runner coordinates observation points and a single interaction
sequence. Extra input such as resize or scroll belongs to an explicit presentation
journey using reusable actions.

| Requirement | Evidence |
| --- | --- |
| Pending indicator is visible | Focused terminal observation while fixtures establish pending work. |
| Spinner animates | At least two distinct relevant indicator states while work remains pending. |
| Required progress layout or styling | Focused semantic/style checks, or a fixed-size snapshot when the whole frame is the requirement. |
| Plain output has no animation or styling/control sequences | Observations throughout the applicable execution, not just its final frame. |
| History survives scroll, resize and return | Explicit interaction and observations of terminal history across the sequence. |

A static screenshot cannot prove animation. Change elsewhere on screen cannot prove
that the spinner changed. A spinner check cannot establish download completion.
Frame snapshots and final model dumps cannot prove transcript preservation. Follow
[terminal evidence boundaries](terminal-harness.md#rendering-assertions-and-snapshots).

## Example: add a mod while observing a spinner

Use the [shared add-mod scenario](bdd.md#one-scenario-in-every-profile). Its product
expectations remain installation, the modlist entry and applicable shared-output and
safety assertions. Do not add a spinner step to that Gherkin.

Attach the spinner check only where a reviewed presentation requirement calls for
an animated pending indicator. TUI Unicode and ASCII can have different representations.
Plain profiles must not animate; their absence-of-animation checks are separate
presentation expectations on the same product journey.

This worked journey describes the required coordination without inventing a
registration API:

1. Set up the compatible installation and deterministic HTTP responses. Register cleanup before launching MMM.
2. The driver starts the add action through the native Go binding using the selected profile's inputs. The scenario-owned fixture signals arrival of the relevant request and holds its response at a controlled pending stage.
3. With pending work independently established, the runner schedules bounded terminal observations. The presentation check observes the relevant indicator and a distinct subsequent state while work is still held. Require a particular glyph sequence only if that sequence is the reviewed requirement.
4. Record the presentation outcome and release the held response whether the check passes, fails or times out. Cleanup also releases it if the journey exits early. A visual failure must not prevent the product journey from continuing when execution remains usable.
5. The driver continues to observable completion. Shared checks verify expected bytes, modlist state, outcome and durable records. Report presentation separately from those product results.

A product-only run does not wait for animation. With the check attached, fixture
coordination creates a bounded observation opportunity without changing the required
product outcome. Use the same payloads and business actions, not another installation
flow just to reach pending work.

Keep the observation window within production timeout/retry behaviour so a delayed
response does not accidentally turn a successful journey into a failure test. Do not
disable production timeouts or retries for visual convenience. Choose pending work
appropriate to the requirement; an unknown total must not acquire an invented percentage.
The [HTTP fixture guide](http-fixtures.md#http-errors-delays-and-transfer-failures)
owns response coordination, request cancellation and handler cleanup.

## Observation timing and execution ownership

Use fixture signals for external work state and supported tui-test observations and
waits for terminal state. Do not sleep for an arbitrary duration and hope an animation
is visible. `WaitIdle` means the screen is quiet, not that work finished; an animating
screen may never become idle.

A driver readiness wait concerns input or progress needed to perform a product action.
The spinner check must not become a hidden precondition of adding a mod. Product
completion must never depend on a particular animation frame.

Coordinate reads and actions in one session sequence. The binding serializes
session operations; a long observation can delay the next action. Use bounded observation
opportunities rather than an independent watcher competing with input or an unbounded
native wait. Keep terminal mechanics in tui-test. This design does not require a custom
PTY, emulator, animation sampler or generic observer framework.

## Lifecycle and results

The runner owns setup and cleanup for the shared invocation. Checks do not launch
duplicate processes or close sessions independently. Stop observations before closing
the session, release held fixtures on every path, retain available evidence and finish
process/fixture cleanup even when a check fails.

Report scenario, profile and check identity with these distinctions:

| Result | Meaning |
| --- | --- |
| Product failure | Capability, authority, data or shared-output expectations were not met. |
| Presentation failure | The applicable experience requirement was not met while it could be observed. |
| Fixture or driver failure | Setup, interaction, observation or cleanup did not provide valid evidence. |
| Not observed / blocked | A prerequisite failed before the requirement could be assessed. |
| Not applicable | The requirement does not apply to this profile, for a documented reason. |

A frozen spinner can fail presentation while installation passes. If installation
fails before pending work starts, do not invent a spinner failure or mark it passed:
retain the product failure and unavailable presentation evidence. A native observation
error is not evidence of successful presentation.

Continue independent product checks after a presentation failure where the process
and evidence remain usable. Otherwise report blocked checks and preserve the original
error. Required product, presentation, fixture and cleanup failures all prevent successful
verification. Optional diagnostic capture failures must not erase the primary failure.
Follow [terminal lifecycle and diagnostics](terminal-harness.md#lifecycle-and-diagnostics).

## When a separate journey is justified

Reuse a journey that already reaches the required state. Spinner animation during add,
progress styling during install and plain-output checks can be assessed without copied
product scenarios.

Use a dedicated presentation journey when the requirement itself needs additional
behaviour: for example, scroll away while work continues, resize, then return to a
pending prompt. Reuse setup, fixtures and actor actions and add only the required
interaction sequence. Follow [feature placement](bdd.md#feature-placement).
Presentation examples are evidence targets, not automatic snapshot baselines; assert
only the reviewed requirement.
