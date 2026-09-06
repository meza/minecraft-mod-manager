# Performance and debugging

Use this reference when a component stalls, rendering costs grow with data, or
you are establishing the project's TUI development checks. Keep the root,
component, and command boundaries in [SKILL.md](../SKILL.md). Performance work
changes the cost of those operations, not who owns them.

## Choose the task guide

Before designing, changing, reviewing, or investigating the work below, read each
applicable guide. A task spanning several concerns requires each relevant guide;
there is no need to reread a guide already loaded for the current work.

| Work | Required guide |
| --- | --- |
| CI, formatting/static checks, race tests, module hygiene, platform builds, or delivery validation | [CI and delivery](ci-and-delivery.md) |
| Slow rendering, profiling, list measurement, syntax preparation, or cache invalidation | [Rendering performance](performance.md) |
| Message traffic, terminal logging, breakpoints, or contributor debugger setup | [Terminal debugging](debugging.md) |
| VHS tapes or terminal demonstrations for documentation/review | [Terminal recordings](recordings.md) |
| Published component-library support requirements or version documentation | [Library support](library-support.md) |

The guides preserve the CI policy, performance examples, diagnostics, and recording
workflow as separate task units. For a new project's delivery setup, read CI and
delivery, plus library support when publishing a component library. Read the
performance, debugging, and recording guides when those concerns enter the task.
