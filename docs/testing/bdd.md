# BDD end-to-end tests

This guide explains how we run BDD scenarios with the godog driver and the shared third-person framework.
Use it when you add or update BDD scenarios under `e2e/features`.

## Quick start

Run the BDD suite:

```bash
make e2e
```

Scenarios live in `e2e/features`, and step definitions live in `e2e/steps_test.go`.

## Writing a scenario

Scenarios are written in the first person. "I" and "me" resolve to the current actor.

```gherkin
Feature: Placeholder BDD framework

  Scenario: I run a placeholder action
    Given I am an actor
    When I perform a placeholder action
    Then I should observe the placeholder outcome
```

Use the actor registry and action types from `testutil/bdd` in your step definitions so outcomes stay observable and reusable.

## Framework building blocks

The `testutil/bdd` package provides:

- Actors that execute actions and record outcomes.
- Actions that stay stateless and reusable across actors.
- Outcomes that represent observable results.
- A shared conversation context for the last actor, action, and outcome.

## Related docs

- `docs/testing/http-vcr.md` for recording and replaying HTTP calls in BDD tests.
- The third-person BDD guidance at https://raw.githubusercontent.com/meza/agent-docs/refs/heads/main/BDD.md explains the actor and action model.
