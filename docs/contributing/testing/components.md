# Testing terminal components and coordination

Use the narrowest test surface that proves the terminal requirement. Component,
render and root-model checks give fast evidence about the target native Bubble Tea v2
architecture. Program-level checks cover runtime behaviour that cannot be established
by direct component or root checks. The [terminal E2E guide](terminal-harness.md) remains the owner
for evidence from a real MMM process in a terminal.

## Choose the evidence boundary

| Requirement | Evidence boundary | What to exercise |
| --- | --- | --- |
| A purpose-specific component operation changes state, returns a result or requests work | Direct component check | Initialise the component, call the relevant operation and assert its state, typed results and explicit effect requests. |
| Text, layout or styling is produced from known state | Direct render check | Render the component at fixed dimensions and capabilities; assert focused semantics or the complete frame when the whole presentation is the requirement. |
| Focus, message routing, child replacement, transcript commits or effect coordination crosses components | Direct coordinating-root check | Exercise the root with real child components and assert routing, ownership, durable events and the effects returned at the boundary. |
| A Bubble Tea program lifecycle, command scheduling or runtime ordering matters | `teatest/v2` program check | Run the smallest native v2 program that contains the behaviour and observe it through the program boundary. |
| A component needs interactive design inspection | Bubblebook story | Open a named story backed by fresh deterministic sample state and inspect the actual rendered component and supported interactions. |
| Shell history, terminal restoration, signals, process exit, scrollback or cross-profile behaviour matters | tui-test E2E | Drive the real MMM process and retain the lifecycle, product and presentation evidence required by the E2E corpus. |

Do not move outward merely because a broader harness already exists. A direct component
check is clearer for a local transition than an in-process program or subprocess. Move
outward when the requirement crosses the narrower boundary. A lower-level check may
supplement broader evidence, but it cannot replace the boundary that owns the behaviour.

## Component and render checks

Keep capability components narrow. A component owns its state, controls, rendering and
typed result; it does not start a program, print directly, perform domain work or own
the whole terminal. Project components expose purpose-specific operations; they do not
need a generic Bubble Tea message and update loop. Call the component operation that owns
the transition and assert its state, result and explicit effect request. Native Bubbles
widgets may retain their framework integration contract; test that contract directly
where it is the behaviour under test.

Render checks begin from deliberate state. Fix width, height, locale, colour policy and
character capability when they affect the result. Prefer focused assertions for labels,
status, selection and important placement. Use a complete render snapshot only when the
reviewed requirement concerns the complete component frame or styling, and inspect every
intentional baseline change.

State-transition checks and render checks can be separate when that makes failures easier
to diagnose. Neither needs a running Bubble Tea program merely to call a component's
purpose-specific operation or render known state.

## Coordinating-root checks

The target native v2 application has one coordinating root. It owns input and focus routing,
component composition, active presentation, durable transcript commits and terminal-level
message routing. Child components return typed results and explicit effect requests to
the root; they do not acquire terminal ownership. The root turns an accepted request into
a Bubble Tea command and returns it to the runtime. The runtime executes that command, and
the root routes its result back to the live component that owns it as a typed result.

Exercise cross-component behaviour at the root boundary with real narrow components.
Check that framework input and command results reach the intended live component, accepted
results return to the owning flow, settled records commit once, active state is removed at
the right time, and the root returns requested commands without duplicating domain
operations. Use small substitutes only for nondeterministic external work and make the
effect request and typed result explicit.

Direct root checks establish model coordination. They do not establish renderer timing,
screen restoration, primary-screen history or OS-signal behaviour.

## Use `teatest/v2` selectively

Use the native Bubble Tea v2 program harness only when the Bubble Tea runtime is part of
the requirement: command scheduling, message arrival order, program startup or shutdown,
or another lifecycle interaction that direct component and root checks cannot prove. Keep the program
under test as small as the requirement permits and use bounded waits based on observable
state.

Do not use an in-process program test as a substitute for direct component or root checks.
It also does not prove real shell integration. Runtime-sensitive terminal behaviour still
needs the real-process E2E boundary when acceptance depends on the operating system,
terminal emulator, process lifecycle or execution profile.

## Stories are inspectable examples

The target component gallery uses the native-v2
[meza/bubblebook](https://github.com/meza/bubblebook) host. Give every meaningful state a
descriptive story name. Construct every story with fresh component and fixture state so
switching away and back cannot retain a previous interaction. Use the same deterministic,
fresh fixture constructors from component, render and root checks in the stories; each
call must return independent mutable state.

Stories render the real component and may add only the thin model adapter needed to host
it. That adapter translates host interaction into the component's purpose-specific
operations; a native Bubbles widget may retain its native message contract. Adapters must
not copy production rendering or business logic. After adding or
changing a story, run the gallery, select the named story, inspect its initial rendering
and exercise its supported interactions. Record automated behaviour in component, render
or root checks; the fact that a story launches is not evidence that its presentation is
correct.

See the [component gallery guide](../../../tools/bubblebook/README.md) for running and
navigating the host.

## Keep real-process E2E distinct

Use tui-test E2E for requirements that only a native MMM subprocess can prove. Preserve
the scenario-owned workspace and HTTP fixtures, profile-specific drivers, shared product
assertions, separately owned presentation checks, bounded observations, diagnostic
capture, exact-session close and cleanup. Do not replace those safeguards with model
output, an in-process Bubble Tea program or a gallery story.

Follow the [BDD architecture](bdd.md) for scenario and driver ownership, the
[presentation guide](presentation.md) for visual evidence, the [HTTP fixture guide](http-fixtures.md)
for deterministic network work and the [terminal harness](terminal-harness.md) for native
process lifecycle and diagnostics.
