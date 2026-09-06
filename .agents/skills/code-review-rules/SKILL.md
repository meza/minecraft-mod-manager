---
name: code-review-rules
description: MUST USE for reviewing Minecraft Mod Manager code changes
---

# Code review

Read [Code review](../../../docs/contributing/reviewing-code/README.md) completely before reviewing a change. Follow its guidance and read every reference it selects for that review before inspecting the change. The document owns the review workflow, scope, findings, verdicts, and verification.

## Select the review lane

For a standalone review, follow the contributor procedure and the response structure below.

For an independent handoff review, follow the assigned lane's scope, evidence restrictions, and reporting format. Report directly to the implementation owner. A documentation-only lane may inspect only the changed documentation and its permitted documentary references, not production code, tests, configuration, or runtime behaviour. Do not run production tests for that lane.

## Retrieve the work item

When the request identifies a Linear ticket, retrieve it through the environment's enabled Linear connector. Extract the ticket identifier from a supplied Linear URL when necessary. If the connector is unavailable, report the missing evidence and its effect on the review.

For a pull request, use the environment's enabled read-only GitHub integration to retrieve its title, complete body, target branch, head revision, and complete discussion history before establishing the comparison. Include general comments, submitted review bodies, inline threads, replies, and available resolved or outdated threads.

Follow every pagination cursor exposed by the integration. Record the pull request identity, target and head revisions, retrieved fields, whether pagination completed, and any unavailable history. Apply the [pull request guide](../../../docs/contributing/reviewing-code/pull-requests.md) when assessing the effect of missing evidence.

Do not read tracker or GitHub credentials from repository or environment files, and do not improvise another access mechanism when an integration is unavailable.

## Use read-only review tools

Discover local changes with read-only status and diff commands. For an unqualified local review, take the union of tracked changes against `HEAD` and untracked files, including additions, modifications, deletions, and renames. Use explicit comparison references and scope limits when supplied by the request, as defined in the contributor guide. Record `HEAD`, any explicit comparison references, and the complete reviewed file set.

Do not modify the submitted implementation, tests, documentation, configuration, tracker state, or Git state. Do not run formatters, auto-fixers, snapshot updates, or generation commands intended to rewrite repository files. Verification may produce ordinary build, test, or coverage output; treat it as a by-product rather than part of the submitted change.

Do not create native GitHub review comments, submit approval or change-request decisions, duplicate findings in pull-request discussion, or close tickets. The implementer owns delivery and ticket decisions.

Run the checks selected by [CONTRIBUTING's Verification section](../../../CONTRIBUTING.md#verification) for the changed surfaces. Capture each command and result. A failed check does not authorise `make fmt`, `make lint-fix`, or another correction command. CI status does not replace required local evidence.

## Return the standalone review

Use these sections in the response, in order:

1. **Context and sources**: requested outcome and rationale, review type, scope, comparison references, reviewed files, and requirement or guidance sources.
2. **Requirements**: acceptance criteria attributed to the request, ticket, or repository document and section.
3. **Advisory verdict**: one of the contributor guide's verdicts with an evidence-based rationale.
4. **Findings**: the supported findings, or `No findings`.
5. **Validation evidence**: local commands and results, targeted inspection evidence, and integration retrieval coverage.
6. **Unverified evidence and questions**: material evidence gaps and decisions requiring input, or `None`.

## Self-verification

Reconcile the work item, every in-scope file and affected flow, applicable guidance, findings being re-reviewed, and verification evidence. Check that integration retrieval is complete or its gaps are disclosed. Apply the contributor guide's verdict precedence, and respect the assigned lane's evidence and reporting limits.
