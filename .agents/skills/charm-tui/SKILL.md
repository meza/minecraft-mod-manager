---
name: charm-tui
description: >
  Design, build, refactor, and review Go terminal interfaces using Bubble Tea,
  Bubbles, Lip Gloss, and Bubblebook. Use whenever work involves Charm TUI
  components, message routing, focus, terminal layout, styling, component stories,
  or UI tests. Teaches one native v2 architecture with a coordinating root,
  reusable components, and isolated Bubblebook stories. Does not cover unrelated
  Charm services or authorize a v1 migration.
---

# Charm TUI

You build terminal interfaces whose components behave the same in the application,
in isolation, and under test. Each asynchronous result returns to the event loop
before changing UI state.

This skill selects one architecture. You compose small reusable components under
one application root and demonstrate them through native v2 Bubblebook stories.

## Working references

Read [Component design](references/component-design.md) when designing, changing,
or reviewing component boundaries, state, routing, or effects. It explains the
selected architecture and routes to the required component lessons for the task.
For design, implementation, review, and investigation, read each applicable
reference below and follow its task routes. Read all applicable guides when the
work spans concerns; do not reload material already read for the current task.

| Work                                                   | Required reference                                                   | Practical material                                                                     |
|--------------------------------------------------------|----------------------------------------------------------------------|----------------------------------------------------------------------------------------|
| Rendering, resizing, themes, or cursor placement       | [Layout and style](references/layout-and-style.md)                   | Frame arithmetic, semantic styles, Unicode, tiny layouts, and cursor composition       |
| Component or application tests                         | [Testing](references/testing.md)                                     | Test layers, controlled commands and time, render fixtures, routing cases, and teatest |
| Stories, fixtures, or catalogue integration            | [Bubblebook](references/bubblebook.md)                               | Package layout, fresh factories, story adapter, state catalogue, and preview lifecycle |
| Slow rendering, runtime diagnosis, or delivery tooling | [Performance and debugging](references/performance-and-debugging.md) | Profiling, bounded list work, cache invalidation, terminal logging, debugging, and CI  |

For a new component, follow the connected path: define its state and operations,
implement bounded rendering, exercise the contract directly, expose fresh stories,
then connect application routing and verify that boundary. The examples are
teaching material to adapt to the project's actual types and dependencies, not a
second framework to copy wholesale.

## Stack and authority

Use the matching major-version family: `charm.land/bubbletea/v2`,
`charm.land/bubbles/v2`, and `charm.land/lipgloss/v2`. The application root returns
`tea.View`. Use `github.com/charmbracelet/x/ansi` for terminal text operations.
Choose exact dependency versions from the project's verified module graph and
supported Go toolchain, rather than copying versions from a research document.

For work in a published component library, read [Library support](references/library-support.md).
That guide
owns the required versioned support matrix, its example, and maintenance rules.

Use [meza/bubblebook](https://github.com/meza/bubblebook) as the native Bubble Tea v2
component catalogue. Read
[Bubblebook stories](references/bubblebook.md) when building, changing, or reviewing
Bubblebook integration, stories, or shared story fixtures.

Read project instructions, architecture, dependency declarations, and the owning
UI code before changing it. This architecture governs new in-scope work; it does
not authorize converting an existing v1 application or replacing an established
architecture during an unrelated fix. Identify conflicts and obtain the needed
scope decision. Do not maintain competing v1 and v2 paths inside a new component.

## One root, explicit components

The application root is the runtime's `tea.Model`. It owns terminal policy and
configuration, navigation, modal precedence, focus allocation, and screen layout.
It receives messages, selects their intended owner, invokes component operations,
and returns commands and the composed view. Keep that coordination readable through
cohesive methods and files; a single runtime root does not mean a single file or
that all domain logic belongs in the root.

Components own their local interaction state and invariants. Expose operations
that express their purpose, such as setting a query, moving a selection, applying
results, focusing, or resizing. Return a typed action or result when the parent
needs to make a decision. A direct synchronous operation does not need to send
itself a message through a command. Pure display elements are render functions
over explicit data, styles, and dimensions.

Keep each mutable value authoritative in one place. Parent-owned data reaches
children as inputs; child-owned editing or selection state is read through its
contract. Do not maintain two independently editable copies. Constructors create
fresh state without I/O. Retained slices and maps need explicit ownership so
multiple instances and fixtures cannot mutate one another accidentally.

Reuse Bubbles for established widget behavior. An encapsulated Bubbles model may
retain its native update contract: the owning component or root forwards the
specific input and lifecycle messages it requires, stores the returned widget
state, and propagates its commands. This is library integration, not permission
to turn every custom pane into another generic message loop. Avoid adapters that
merely rename every Bubbles method without protecting a real component contract.

Application components do not implement `tea.Model` just to become reusable.
The development story adapter implements it to host a component in Bubblebook.
That adapter is a boundary to the catalogue, not a second production component
architecture. Component code never imports Bubblebook or its story packages.

## Input, lifecycle, and effects

Route user input to the active modal first, then to its intended focused control
when no modal owns it. Modal input must not reach underlying controls. Any
application-wide escape or quit policy is explicit, including its precedence
over a modal. Focus allocation has one owner; children reflect that allocation.
Mouse input uses the actual layout regions and local coordinates.

Focus gates user interaction, not every message. In-flight results, spinner ticks,
cursor messages, and other widget-specific messages still reach their live owner
as required. Tag or otherwise identify instance-specific work. Unrelated children
must not consume each other's results. Resize and theme updates are deliberately
distributed to affected components, rather than broadcast as arbitrary input.

Construction, initial sizing, activation, focus, and deactivation have explicit
ordering. Start activation-owned work once per activation. Later commands follow
explicit user actions or result messages, including retries and recurring widget
ticks. Preserve commands returned from initialization, focus, resize, and widget
updates. When an owner is replaced or disposed, cancel work that supports
cancellation and stop scheduling its recurring work. Reject stale results using
instance or request identity, since cancellation alone cannot prevent an already
completed result arriving.

`Update` makes cheap state transitions and creates commands. Commands perform
I/O or expensive computation against captured inputs and injected dependencies,
then return typed result messages. They do not mutate component or root state
from their goroutines. Rendering performs no I/O and starts no work.

Use `tea.Batch` for independent effects. When a later action depends on a result,
handle the result message and launch the next action only when its preconditions
hold. In particular, saving must succeed before a success-dependent close.
`tea.Sequence` orders execution; it is not a substitute for checking errors or
waiting for the update loop to apply a result. Loading, failure, retry, and stale
result behavior belong to the visible state contract.

## Rendering and geometry

Compose string-based component output with Lip Gloss into the root's `tea.View`.
Do not introduce a screen-buffer framework, renderer abstraction, or cache without
an established requirement that this chosen path cannot satisfy.

Styles are injected values with semantic roles such as selected, muted, error,
and border. Theme and background detection happen at the application boundary.
Components receive resolved styles, not an application-wide service locator or
hidden terminal-environment dependency. State must remain understandable without
color alone.

The parent allocates each child's outer rectangle. A component owns its internal
decoration and derives its usable content area from that allocation and its
actual style frames. Apply that subtraction once at the owning boundary. Clamp
dimensions to nonnegative values and define behavior when even the decoration
cannot fit. Resizing changes geometry without losing the user's data or creating
unintended effects. A style width is not a universal clipping guarantee.

Measure and truncate display cells with ANSI-aware helpers, not byte lengths or
string slicing. Account for borders, padding, margins, prompts, and scrollbars.
Rendering must remain bounded at tiny sizes and with long styled Unicode text.

Strings do not carry terminal cursor metadata. For controls using a real cursor,
expose the needed local cursor information alongside their content. Each owning
composition boundary translates coordinates exactly once, clips or hides an
out-of-bounds cursor, and selects only the focused visible cursor for its view.
Keep view metadata meaningful when the component is hosted in Bubblebook too.

Given the same state, styles, dimensions, and explicit environment inputs,
rendering produces the same content and metadata. Keep costly preparation out of
the render path. Optimize measured hot paths and make cache dependencies and
invalidation explicit rather than caching every renderer preemptively.

## Stories and tests share the component

Every new reusable visual component gets named Bubblebook stories for its
meaningful states. Fresh fixture constructors provide data, styles, dimensions,
and controlled dependencies to both stories and tests. Production code uses the
component, never the fixture or catalogue. A story adapter translates catalogue
events and composes the component's actual output; it does not reimplement its
behavior. See [Bubblebook stories](references/bubblebook.md) for hosting.

Most checks run directly through stable component interfaces: state transitions,
returned actions, and fixed-size rendering. Test the parent separately for modal
precedence, focus, routing, resize allocation, and asynchronous result ownership.
Exercise the real command loop only for behavior that needs the runtime, using
the project's pinned `github.com/charmbracelet/x/exp/teatest/v2` dependency and a
small shared setup for terminal options. Keep that experimental dependency in
test support; do not build a general testing framework around it.

Choose assertions by the component's contract. Cover fresh-instance isolation,
empty and populated state, relevant loading/error/retry paths, stale results,
focus transitions, repeated rendering, and tiny or resized layouts. Render
assertions pin terminal size, color profile, theme, data ordering, and any time or
random inputs that affect output. Drive asynchronous tests with messages and
controlled dependencies instead of sleeps. Goldens prove presentation; pair them
with behavioral assertions and inspect intentional updates.

Run relevant Go tests and project formatting/static checks. Include race-enabled
tests for effectful behavior and builds on the project's supported platforms.
Report an unavailable race toolchain rather than silently treating an ordinary
test run as equivalent. Bubblebook inspection complements automated assertions;
a story or VHS recording is not a correctness oracle. Record demos only when
they serve the requested documentation or visual review.

## Delivery and self-verification

Deliver the requested component or application change with its relevant tests,
stories, and usage documentation. Explain state and input ownership, dependency
versions actually verified, and checks performed. Scale the explanation to the
change.

Before handoff, verify that:

- The chosen v2 stack matches the project's verified dependencies.
- Application routing, state, layout, and effect ownership each have a clear home.
- Bubbles commands and required non-input messages survive the integration.
- Effects cannot mutate UI state concurrently or apply to a replaced owner.
- Rendered content and cursor metadata fit the allocated region.
- Stories use fresh instances of the same production component and test fixtures.
- Production packages have no dependency on Bubblebook, stories, or fixtures.
- Tests prove relevant behavior and presentation under controlled conditions.
- Unverified story execution is reported as incomplete, not silently substituted.
- Changes stay within the user's scope and the project's target architecture.

## Primary references

Use the project's pinned source for exact APIs. These upstream references explain
the selected ecosystem; they do not authorize changing the project's versions:

- [Bubble Tea v2 migration guide](https://github.com/charmbracelet/bubbletea/blob/main/UPGRADE_GUIDE_V2.md)
- [Bubbles components](https://github.com/charmbracelet/bubbles)
- [Lip Gloss](https://github.com/charmbracelet/lipgloss)
- [Crush UI ownership and composition](https://github.com/charmbracelet/crush/blob/main/internal/ui/AGENTS.md)
- [Teatest v2 source](https://github.com/charmbracelet/x/tree/main/exp/teatest/v2)
