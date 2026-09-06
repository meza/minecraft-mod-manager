# Charm TUI

Charm TUI guides an agent building Go terminal interfaces with Bubble Tea v2,
Bubbles v2, Lip Gloss v2, and Bubblebook. It teaches one architecture: a root
coordinates the application, components own their local behavior, and thin story
adapters expose the same components in the catalogue.

Ask the agent to build, refactor, or review a Charm interface, for example:

> Build a searchable results component with loading and error states, tests,
> and Bubblebook stories for my Charm v2 application.

The skill covers input and focus ownership, asynchronous work, terminal geometry,
semantic styles, deterministic tests, and isolated component development. It
respects the existing project's scope and does not automatically migrate a v1
application.

## Bubblebook stories

Use [meza/bubblebook](https://github.com/meza/bubblebook), the native Bubble Tea v2
catalogue, to inspect fresh instances of the same
components used by the application. Stories share controlled fixture inputs with
component tests.

The [story reference](references/bubblebook.md) explains fixture ownership, the
preview lifecycle, and verification. The agent's complete instructions are in
[SKILL.md](SKILL.md).

## Worked guidance

The skill pairs its architectural rules with practical references:

| Reference | What you can learn and apply |
| --- | --- |
| [Component design](references/component-design.md) | Contracts and comparisons, with task routes to the complete Results example, input routing, effects and ordering, and lifecycle/Bubbles lessons |
| [Layout and style](references/layout-and-style.md) | Calculate outer and content sizes; inject semantic styles; handle ANSI and Unicode; position a real cursor through nested geometry |
| [Testing](references/testing.md) | Select the applicable layers, then read the component, effects/time, rendering/golden, parent-routing, or runtime test guide |
| [Bubblebook stories](references/bubblebook.md) | Share fixtures, build a thin story adapter, register fresh factories, inspect edge states, and preserve preview lifecycle behavior |
| [Performance and debugging](references/performance-and-debugging.md) | Select CI/delivery, rendering performance, terminal debugging, or recording guidance for the current task |
| [Library support](references/library-support.md) | Maintain a published component library's versioned Go and dependency support contract |

Start with component design and follow the references for the task at hand. The
tables explain the selected approach; the diagrams and code show how its parts
connect. All examples use the same root-and-components architecture. The task
routers require every guide applicable to the work; related examples retain
explicit prerequisite links. Each lesson keeps its code, assumptions, and
verification guidance together.

Install through this repository's [marketplace instructions](../../../../README.md).
Repository changes follow the [contribution guide](../../../../CONTRIBUTING.md).
