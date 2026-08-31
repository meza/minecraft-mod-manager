# Pull request reviews

Use the environment's enabled read-only GitHub integration to retrieve the current pull request
title, complete body, target branch, head revision, linked work item, and complete discussion history
before establishing the review baseline. Include general comments, submitted review bodies, inline
threads, replies, and resolved or outdated threads when they remain available. Do not read GitHub
credentials from repository or environment files or improvise another access mechanism.

Follow every pagination cursor exposed by the integration. Under `Context and sources`, record the
pull request identity, target and head revisions, retrieved fields, whether pagination completed,
and any history the integration reports as unavailable.

If the integration or material history is unavailable, record exactly what could not be retrieved.
The canonical verdict precedence applies: use `Review incomplete` when a missing portion creates a
material evidence gap, including when it prevents the reviewer from establishing requirements,
scope, comparison endpoints, or the state of a prior finding. A missing portion is non-material only
when the available evidence independently establishes the affected contract; explain that basis in
`Validation evidence`.

Use that material as evidence of purpose, explicit boundaries, known constraints, prior findings,
implementation decisions, and questions already resolved. It cannot override the work item's
requirements, accepted architecture, or current repository guidance. A different implementation or
additional change is not a finding by itself; report only a concrete adverse consequence.

Pull request reviews use the same root `code-review.md` artifact and advisory verdicts as every other
review. Do not create native GitHub review comments, submit an approval or change-request decision,
or duplicate findings in pull-request discussion. Humans decide how the advisory review affects the
pull request.

Inspect the pull request's full declared review surface even when an earlier comment highlights a
smaller area. A user may explicitly narrow the review; record that boundary and do not claim
coverage outside it.

Run every required local gate from the root contribution guide and record each command and result in
`code-review.md`. CI results may provide additional context, but they do not replace local evidence
and do not determine the advisory verdict by themselves.
