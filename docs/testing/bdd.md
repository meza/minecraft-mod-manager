# BDD end-to-end tests

Product-level terminal scenarios use Godog as the Gherkin runner, `testutil/bdd` as the actor/action/outcome vocabulary, and tui-test as the terminal driver.

## Run the scenarios

```bash
make e2e
```

This command builds a host-native MMM executable with the `e2e` build tag, then runs the tagged Godog suite. Ordinary `make test` does not build the E2E binary or require tui-test.

Feature files live in `e2e/features`. The Godog runner and reusable step definitions live in the `e2e` package.

## Write product conversations

Write scenarios in third person. Name the actor, describe an action, and assert an observable outcome. This example expresses the target cancellation contract; step wording is illustrative, not a claim that the steps or behavior are already implemented:

```gherkin
Scenario: A user cancels initialization before choosing a loader
  Given Alice uses MMM in an empty workspace
  When Alice starts interactive initialization
  Then Alice should see the i18n key "cmd.init.prompt.loader.question"
  When Alice cancels initialization
  Then Alice should observe an interrupted outcome
  And Alice should find no configuration in the workspace
```

Use the generic actor, action, outcome, conversation-context, and pronoun infrastructure from `testutil/bdd`. Keep product-specific reusable actions and outcomes in `e2e`. Step definitions translate the conversation into those actions; the action delegates terminal work to the tui-test adapter.

## Assertion boundary

Scenarios run a separately built MMM process in a temporary workspace. They may assert:

- i18n keys and interpolation arguments requested by MMM;
- terminal text or state reported by tui-test;
- exit status;
- observable filesystem effects.

Do not assert internal Bubble Tea messages, renderer buffers, polling behavior, ANSI-normalized byte streams, or other implementation details.

The E2E process receives `MMM_TEST=1` so localization expectations use stable keys rather than translated wording. The value is conventional; the presence of `MMM_TEST` enables the mode. This mode is available to Go test binaries and `e2e`-tagged binaries only; release builds continue to render localized text.

## Adding coverage

For HTTP-dependent scenarios, follow the agreed [HTTP fixture design](http-fixtures.md): Godog owns local Go HTTP servers, while an E2E-tagged MMM process receives test-only endpoint overrides. This wiring is not implemented yet. Keep fixture setup and request verification in the scenario's supporting actions and lifecycle; Gherkin should describe product behavior. Fixture failures must fail independently of the product's response.

Framework-only scenarios are not product coverage. Establish product coverage from the scenarios and observations that exercise [intent's acceptance journeys](../intent.md#acceptance-and-evidence), not from the existence of the harness.

For each new product scenario:

1. review and agree the product requirement;
2. write the Gherkin conversation;
3. add or reuse actor actions and observable outcomes;
4. drive the native MMM process through tui-test;
5. remediate production behavior when the scenario exposes inconsistency.

The [legacy terminal test ledger](legacy-terminal-test-ledger.md) can help find historical intent, but it must not be treated as the desired result. See the [tui-test E2E guide](terminal-harness.md) for adapter and lifecycle details.
