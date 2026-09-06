# BDD architecture and shared product journeys

This guide owns scenarios, profile execution, driver responsibilities and product
assertions. Start with the [testing overview](README.md). [Product intent](../../intent.md#acceptance-and-evidence)
owns expected behaviour; [presentation testing](presentation.md) owns visual checks.

## One scenario in every profile

Write the capability once, without frontend mechanics. This is a human-readable
worked journey, not an API reference:

```gherkin
Scenario: Alice adds a mod to her installation
  Given Alice has an installation compatible with Sodium
  When Alice adds Sodium to her installation
  Then Alice should find Sodium installed
  And Alice should find Sodium in her modlist
```

The runner executes this unchanged scenario in all five
[execution profiles](../../intent.md#execution-modes-and-operator-intent). Selection
belongs outside shared Gherkin: no duplicate scenarios, profile-specific steps or
profile Examples table. Each run has isolated actor/conversation state, workspace,
fixtures and process lifecycle. Reports identify both scenario and profile.

| Profile | Driver translation of the same action |
| --- | --- |
| Explicit unattended / pure CLI | Supply required arguments and policies with `--unattended`; never answer prompts. |
| Interactive TUI with Unicode | Supply inputs and operate the required rich controls. |
| Interactive TUI with ASCII | Perform equivalent interaction with supported ASCII presentation. |
| Plain interactive | Supply inputs and answer line-based questions without relying on rich controls. |
| Non-interactive | Supply required decisions and arrange non-interactive I/O conditions without answering prompts. |

All drivers use tui-test's native Go binding for supported process and terminal operations.
Supply equivalent decisions in every profile. Complete arguments alone do not select
unattended execution. Shared assertions still require installed mod bytes and the
modlist entry; one profile's success cannot stand in for another's evidence.
Network-dependent runs use the [HTTP fixture design](http-fixtures.md).

## Runner, drivers and shared assertions

```text
Runner selects profile and applicable checks
  Gherkin -> Godog step -> actor action -> selected driver
    -> tui-test Go binding -> Rust engine -> real MMM process
  Observed evidence -> shared product assertions
                    -> attached presentation checks, where applicable
```

| Owner | Responsibility |
| --- | --- |
| Gherkin | Describe actors, actions and observable product outcomes. |
| Runner / scenario lifecycle | Select profiles and checks; isolate state; coordinate execution and cleanup; report results. |
| Shared steps and actor actions | Express product actions independently of interaction mechanics. |
| Mode-specific driver | Perform actions and expose product-visible evidence through the binding. |
| Product assertions | Evaluate capability, authority, persisted effects and durable-output expectations. |
| Presentation checks | Evaluate applicable presentation requirements from the same journey's observations. |
| HTTP fixtures | Control external responses and report independent fixture failures. |

Drivers contain neither business-outcome assertions nor presentation assertions.
They do not decide that installation is correct, a spinner animates or a layout
matches. They return observations and execution errors. They must allow an action
to be attempted even when the product is expected to reject it.

Readiness waits and execution errors are not acceptance assertions. A driver may
wait for an input control needed to enter a value and report failure to reach it.
It must not wait for a decorative spinner frame before continuing or declaring
completion. Use observable readiness, process completion and product effects
appropriate to the action. A quiet screen alone does not establish completion.

Use the existing actor/action/outcome and conversation vocabulary from `testutil/bdd`.
Keep steps thin: resolve actors, compose actions and assert outcomes. Keep conversation
context limited to the subject, actor, action and outcome needed to resolve references;
do not hide scenario state in step globals. These are responsibilities, not prescribed
Go interface names or a custom terminal abstraction. tui-test owns terminal machinery.

## Product and presentation assertions

Classify a check by the requirement it proves, not the API used to observe it.
Reading terminal text does not automatically make a check a visual test.

| Product checks | Presentation checks |
| --- | --- |
| Expected file bytes and modlist/lockfile effects | Spinner changes while work remains pending |
| Rejection, authority and no-prompt policy | Required progress styling and layout |
| Exit outcomes and recovery effects | Focus styling, clipping and translated wrapping |
| Shared durable records and transcript parity | Absence of animation and styling/control sequences in plain output |

Required results and decision records are product evidence even when observed in
the terminal. Their text and formatting follow the shared
[transcript contract](../../intent.md#active-display-and-permanent-transcript).
Equivalent outcomes and decisions use the same durable records under matching
locale and character capabilities, whether supplied by prompts, arguments or defaults.
Records appear once as outcomes settle, without replay or manufactured questions.
Do not normalize away genuine differences, missing records, duplicates or ordering.

Use terminal observations, exit status and filesystem effects, not internal Bubble
Tea messages, renderer buffers or model state. Stable i18n keys and arguments verify
message selection; actual translations are needed for translated presentation.
See [localization expectations](terminal-harness.md#stable-localization-expectations).

Presentation checks leave the scenario and its business assertions unchanged. They
attach to the same invocation with separate outcomes. The
[spinner example](presentation.md#example-add-a-mod-while-observing-a-spinner)
explains timing and failure handling.

## Feature placement

Shared capability features belong directly in `e2e/features`. Reusable actions and
assertions belong in E2E test code. Keep journeys reusable by presentation checks.
Use these feature subfolders only for requirements intrinsic to that experience,
creating them when needed:

| Folder | Dedicated experience |
| --- | --- |
| `tui/` | Rich interaction common to both character profiles. |
| `tui/unicode/` | Requirements specifically about Unicode presentation. |
| `tui/ascii/` | Requirements specifically about ASCII alternatives. |
| `plain-interactive/` | Line-based interaction without control sequences. |
| `unattended/` | Explicit no-prompt execution regardless of terminal capabilities. |
| `non-interactive/` | Interaction unavailable, including redirected I/O. |

Spinner animation does not require another add-mod feature in `tui/`. Attach its
check to a journey that reaches pending work. A dedicated presentation journey is
justified when its requirement needs additional actions, such as scrolling away,
resizing and returning. Reuse setup and actions; explain the reason in the feature.

Write scenarios in third person with named actors and unambiguous references.
Cancellation before answering an interactive question is profile-specific; cancelling
ongoing work can be shared where the same product contract applies. Different test
mechanics alone do not justify an exception.

## Run and extend coverage

Use `make e2e` after completing the [terminal prerequisites](terminal-harness.md#prerequisites).
The [suite guide](terminal-harness.md#running-the-suite) owns binary selection and
commands. Godog must fail verification when a step is undefined or pending.

For a requirement, establish its product contract, write or reuse the scenario and
actor actions, arrange deterministic fixtures and drivers, and assert shared product
outcomes. Attach presentation checks where a reviewed requirement calls for them.
Exercise the relevant profiles and retain separate scenario/profile/check results.

Fixture errors must not pass as expected product rejection. Do not weaken assertions
to accommodate implementation gaps. Required presentation checks are part of acceptance
even when product checks pass. Follow [lifecycle and results](presentation.md#lifecycle-and-results)
for cleanup, reporting and unavailable evidence.
