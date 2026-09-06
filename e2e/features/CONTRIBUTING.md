# Contributing BDD features

Write capabilities in third person using named actors, product actions and
observable outcomes. Shared features must be frontend agnostic: no prescribed
widgets, layout, colours, keystrokes, spinner frames or process exit codes.

Assert success, rejection, incompatibility, interruption and recovery through
visible operation results and their effects. An interactive TUI operation has no
exit code. Numeric exit-code mappings belong only in CLI-specific checks for
unattended, non-interactive and plain-interactive execution; keep them out of
shared steps and Examples columns. See [assertion ownership](../../docs/contributing/testing/bdd.md#product-and-presentation-assertions).

Use the agreed [actors and pronouns](../../docs/contributing/testing/bdd.md#actors).

Write each shared scenario once. The runner executes it unchanged in every
profile; keep profile selection outside steps and Examples tables. See the
[add-mod example](../../docs/contributing/testing/bdd.md#one-scenario-in-every-profile).

Keep shared capability features directly in this folder. Use profile subfolders
only for requirements intrinsically about that experience, following the
[feature placement rules](../../docs/contributing/testing/bdd.md#feature-placement).
Different driver mechanics do not justify copied scenarios.

Presentation checks reuse existing product journeys where those journeys reach
the required state. Do not add a spinner assertion to shared Gherkin or write
another add-mod feature just to observe a spinner. Follow
[presentation testing](../../docs/contributing/testing/presentation.md) for attachment, timing
and results, and the [E2E contribution guide](../CONTRIBUTING.md) for fixture,
harness and verification requirements.
