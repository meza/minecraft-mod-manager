# Contributing BDD features

Write capabilities in third person using named actors, product actions and
observable outcomes. Shared features must be frontend agnostic: no prescribed
widgets, layout, colours, keystrokes or spinner frames.

Write each shared scenario once. The runner executes it unchanged in every
profile; keep profile selection outside steps and Examples tables. See the
[add-mod example](../../docs/testing/bdd.md#one-scenario-in-every-profile).

Keep shared capability features directly in this folder. Use profile subfolders
only for requirements intrinsically about that experience, following the
[feature placement rules](../../docs/testing/bdd.md#feature-placement).
Different driver mechanics do not justify copied scenarios.

Presentation checks reuse existing product journeys where those journeys reach
the required state. Do not add a spinner assertion to shared Gherkin or write
another add-mod feature just to observe a spinner. Follow
[presentation testing](../../docs/testing/presentation.md) for attachment and evidence,
and the [E2E contribution guide](../CONTRIBUTING.md) for implementation requirements.
