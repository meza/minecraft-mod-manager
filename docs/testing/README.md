# Testing product capabilities and presentation

This guide is for contributors writing or maintaining MMM's end-to-end tests.
[Product intent](../intent.md) defines required behaviour; these guides explain
how tests obtain and assess evidence. The [glossary](../GLOSSARY.md) owns vocabulary.

## Choose the guide for your task

| Task | Canonical guide |
| --- | --- |
| Write scenarios; understand runner, actions, drivers and product assertions | [BDD architecture](bdd.md) |
| Check spinners, progress, layout or styling without copying journeys | [Presentation testing](presentation.md) |
| Set up tui-test, run tests, manage processes and capture terminal evidence | [Terminal harness](terminal-harness.md) |
| Control HTTP responses, downloads and pending work across the process boundary | [HTTP fixture design](http-fixtures.md) |
| Understand the existing in-process recording helper | [HTTP VCR reference](http-vcr.md) |
| Consult historical terminal-test evidence | [Legacy terminal test ledger](legacy-terminal-test-ledger.md) |

The VCR helper and historical ledger do not define E2E fixtures or current acceptance
requirements. Contribution rules live in the [E2E](../../e2e/CONTRIBUTING.md) and
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

## Available tooling and target architecture

The available suite uses the tui-test CLI adapter. Go-binding drivers, the complete
profile matrix and attached presentation checks described here are target architecture,
not implemented APIs. HTTP fixture wiring is also pending. Use the
[current prerequisites](terminal-harness.md#current-cli-prerequisite) and
[suite commands](terminal-harness.md#running-the-suite) to run available tests.

No separate presentation command or registration API is defined here. Implementation
must preserve the documented suite entry point and real MMM process boundary.
