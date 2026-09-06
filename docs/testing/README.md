# Testing product capabilities and presentation

This guide defines MMM's terminal testing contract for contributors writing,
running or maintaining tests.
[Product intent](../intent.md) defines required behaviour; these guides explain
how tests obtain and assess evidence. The [glossary](../GLOSSARY.md) owns vocabulary.

## Choose the guide for your task

| Task | Canonical guide |
| --- | --- |
| Verify a documentation-only change | Root [verification guidance](../../CONTRIBUTING.md#verification) |
| Choose component, render, root, runtime, story or real-process evidence | [Components and coordination](components.md) |
| Set up the native binding and run the suite | [Terminal harness](terminal-harness.md#prerequisites) |
| Write scenarios; understand runner, actions, drivers and product assertions | [BDD architecture](bdd.md) |
| Check spinners, progress, layout or styling without copying journeys | [Presentation testing](presentation.md) |
| Investigate a failure, manage processes and capture terminal evidence | [Lifecycle and diagnostics](terminal-harness.md#lifecycle-and-diagnostics) |
| Control HTTP responses, downloads and pending work across the process boundary | [HTTP fixtures](http-fixtures.md) |

Contribution rules live in the [E2E](../../e2e/CONTRIBUTING.md) and
[feature](../../e2e/features/CONTRIBUTING.md) guides.

## One journey, separately owned checks

The runner executes a shared capability scenario in every execution profile. A
selected driver performs interactions through tui-test's Go binding and exposes
evidence. Product assertions evaluate the capability; presentation checks can
observe the same process execution and evaluate its experience requirements.

An add-mod journey can establish both that a mod was installed and that its pending
indicator animated. Installation belongs to product assertions; animation belongs
to presentation checks. Neither assertion belongs inside the driver. A required
presentation failure still fails verification even when installation succeeds.

## Historical references

The [testing archive](archive/README.md) preserves the VCR reference and removed-test
ledger for historical investigation. They do not define E2E fixtures or acceptance
requirements.
