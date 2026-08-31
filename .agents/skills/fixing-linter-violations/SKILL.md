---
name: fixing-linter-violations
description: MUST USE when fixing linter violations, reports, errors, warnings.
---

# Rules

Adhering to these rules is mandatory. No exceptions.

## First Rule

Every individual suppression must be authorized by the user AT EVERY TIME. No assumed authorization. Ever.

## Second Rule

The linter is diagnostic evidence, not the acceptance target. A finding means the underlying boundary, responsibility, data shape, resource lifecycle, or test design is wrong; making the diagnostic disappear without correcting that cause is failure.

## Third Rule

Gaming the linter, pursuing passing linters instead and ignoring the [Second Rule](#second-rule) is a critical failure.

## Fourth Rule

The user has the authority to override any of these rules. That override must come in a clear, explicit command with no room for misunderstanding. This authorization only lasts for the given topic you're working on in a given session. It does not transfer between sessions and tasks. You can never assume, presume or apply the authorization otherwise.
