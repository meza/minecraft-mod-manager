# Code review

Code review determines whether a change satisfies the requested outcome and fits the repository's current engineering and architectural contracts. Reviews are evidence-based and advisory: the reviewer reports every supported observation, while the implementer decides whether a finding prevents delivery.

Review the submitted work without changing it. Leave corrections, Git changes, and ticket closure to the implementer.

## Choose the review

Read the guidance for the review you are doing before examining the change:

- [Initial review](./reviewing-code/initial-review.md) covers the complete change.
- [Re-review](./reviewing-code/re-review.md) covers findings, fixes, and affected flows. Apply initial-review guidance to new, expanded, or previously unreviewed work.
- [Pull request reviews](./reviewing-code/pull-requests.md) covers the pull request description and discussion.
- [Architectural fitness](./reviewing-code/architectural-fitness.md) applies to changes involving executable behaviour, schemas, public or typed contracts, module boundaries, dependencies, shared components, business rules, or architecture documentation.

Architectural review is not needed for prose, formatting, generated output, repository metadata, or mechanical changes that involve no architectural choice.

Read the relevant package guides and contribution requirements, including the Go and testing guidance when those areas change.

## Understand the change

Start with the requested outcome, its rationale, intended behaviour, constraints, and acceptance criteria. Read the associated ticket and its references when there is one. Otherwise, use the review request. Identify which requirement comes from which source, and raise unclear or unavailable requirements rather than guessing.

The request, ticket requirements, normative references, and repository guidance define what the change must satisfy. Implementer notes can explain supporting work or add compatible constraints, but cannot narrow those requirements. Cleanup, refactoring, or supporting work is not a defect merely because it was not explicitly requested; a finding needs a concrete adverse consequence.

## Agree the scope

Unless the request specifies otherwise, an initial review covers the entire change:

- For local work, compare the working tree with `HEAD`, including staged, unstaged, and untracked files.
- For a pull request, compare its head with the merge base of its target branch.
- Use an explicit pair of comparison references when the request supplies one.

Include additions, modifications, deletions, and renames. Note the revisions and files reviewed so the scope is clear. Git status and diffs help establish this scope; staging, tracking, branch, and commit hygiene are not review findings.

For a limited review, state what is included and excluded. Examine every file and materially affected flow within that scope, and do not claim coverage beyond it. If the comparison cannot be established, explain what is missing and mark the review incomplete.

## Assess the work

Judge new and materially changed work against current repository guidance and configuration. Neighbouring legacy code can explain constraints, but does not establish an acceptable pattern. A narrow change need not repair unrelated debt; copied, extracted, substantially rewritten, or newly exposed code must meet the current standard.

Trace the actual control and data flows. Consider correctness, architecture, security, compatibility, failure handling, recovery, side effects, cross-platform behaviour, and test effectiveness where relevant. Tests should assert material behaviour, not just execute code.

Report observations introduced, worsened, newly exposed, or made directly relevant by the change. Each finding needs:

- the concrete observation and its location;
- the reachable condition, execution path, or governing evidence;
- the consequence for users, data, security, operations, architecture, or maintenance; and
- a bounded recommended outcome.

Keep findings objective and unranked, without severity, priority, or delivery-impact labels. Group occurrences with the same root cause and identify every affected location. Report every supported finding, even when another finding already warrants changes.

Exclude unrelated debt, personal preference, hypothetical risks without a credible path, and redesigns whose only benefit is aesthetic consistency. A defect demonstrated by a check is a finding: report it once with the command and result, adding any analysis needed to explain the consequence. An acceptable change needs no findings.

## Verify each changed surface

Follow the contribution guide's [Verification](../CONTRIBUTING.md#verification) requirements for every changed surface. Record the commands, results, and relevant inspection evidence.

- For production behaviour, use tests through stable interfaces at the appropriate level, together with the required code-quality, security, build, and end-to-end checks.
- For documentation, check content, links, rendering, or other documentation-specific evidence.
- For configuration, use parser, schema, or tool-native validation.
- For mechanical edits, use searches, diffs, and relevant static checks.
- For generated output, use the owning generation or staleness check without updating the output.

Combine the required checks for mixed changes. Production tests are not a substitute for the appropriate documentation, configuration, generated-output, or mechanical checks. CI can provide additional evidence, but does not replace required local verification.

Run checks rather than correction commands such as `make fmt`, `make lint-fix`, snapshot updates, or regeneration. Ordinary build, test, and coverage output is a by-product of verification, not part of the submitted change. Leave fixes to the implementer.

If a required check cannot run, record the attempted command, the exact failure or constraint, and what remains unverified. A failed check that conclusively demonstrates a defect is evidence for a change recommendation; it does not by itself make the review incomplete.

## Complete the review

Explain what was reviewed, the requirements and sources used, the findings, the verification results, and any outstanding questions or evidence gaps. Make each finding actionable and self-contained.

Use one advisory verdict with a brief explanation:

- `Review incomplete` when a material evidence gap remains. This takes precedence over the other verdicts.
- `Changes recommended` when findings remain.
- `No changes recommended` when the review is complete and no corrections are needed.

Before finishing, account for every in-scope file and affected flow, every applicable requirement, all findings being re-reviewed, and the required verification. The implementer decides how the review affects delivery.
