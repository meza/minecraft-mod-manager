---
name: code-review-rules
description: MUST USE for reviewing Minecraft Mod Manager code changes
---

# Code review router

## Load the canonical policy

Read [`../../../docs/code-review.md`](../../../docs/code-review.md) completely before reviewing a
change. It owns the review workflow, scope, evidence, verdicts, and output format. Read every
reference that it selects for the current review before inspecting the change.

## Establish the work item

- When the request identifies a Linear ticket, retrieve it through the environment's enabled
  Linear connector. Extract a ticket identifier from a supplied Linear URL when necessary.
- Do not read tracker credentials from repository files or environment files, and do not improvise
  another access mechanism. If the connector is unavailable, record the missing evidence under
  `Unverified evidence and questions` in `code-review.md`.
- When no ticket is identified, use the request's rationale, intended behaviour, constraints, and
  acceptance criteria as the work item. Record material ambiguity instead of inventing a
  requirement.
- A ticket's requirements and normative references cannot be narrowed by implementer notes.
  Non-conflicting implementer constraints and justified additional work remain part of the review.

## Preserve the review boundary

- The only review artifact is `code-review.md` in the repository root. Put all findings, evidence,
  questions, and the advisory verdict there.
- Do not modify submitted implementation, tests, documentation, configuration, or tracker state
  during a review. Writing the required root `code-review.md` artifact is the only exception. Do not
  delete `code-review.md`.
- Do not run formatters, auto-fixers, or other commands intended to rewrite repository files.
- Do not mutate Git state. Read-only Git commands may be used to discover and understand the review
  surface, but staging, tracking, branch, and commit state are not review findings.
- Verification commands may generate ordinary build, test, or coverage artifacts. Treat those as
  review by-products, not as part of the submitted change.
- The reviewer cannot close tickets or decide whether a finding blocks delivery. Humans own those
  decisions.

## Complete the review

- For an initial review, cover every file and materially affected control or data flow in the
  declared review surface. A user may explicitly narrow that surface; record the boundary and do
  not claim coverage outside it.
- For a re-review, use the prior matching `code-review.md` as the baseline and follow the bounded
  re-review guidance selected by the canonical policy.
- Read [`../../../CONTRIBUTING.md`](../../../CONTRIBUTING.md) and run every required gate command
  named in its `Required local checks` section. Record each command and result in `code-review.md`.
  The contribution guide's advice to run `make fmt` or `make lint-fix` after a failed gate is for
  implementers, not reviewers; record the failure without running either fix target. Never
  substitute a CI status for required local evidence.
- Finish by reconciling the work item, every in-scope file, affected flows, applicable guidance,
  prior findings when relevant, and validation evidence. If evidence is incomplete, use the
  `Review incomplete` advisory verdict.
