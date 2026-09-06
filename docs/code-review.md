# Code review

Code review determines whether a change satisfies the requested outcome and fits the repository's
current engineering and architectural contracts. Reviews are evidence-based and advisory: the
reviewer reports every supported observation, while humans decide whether a finding prevents
delivery.

A standalone review is recorded only in `code-review.md` at the repository root. Do not publish
its findings through native pull-request review controls or another parallel review artifact.

An independent handoff review required by repository coordination policy is a separate review lane,
not another standalone review or a new human approval gate. Its reviewer reports directly to the
implementation owner in the format requested by that policy and does not create or replace the root
artifact. When that lane is limited to documentation, review only the changed documentation and its
governing documentary evidence. Do not inspect production code, configuration, or runtime behaviour,
and do not run production software tests. These restrictions do not narrow a standalone review.

The remainder of this document governs standalone reviews.

## Load the guidance for this review

Read this document completely for every standalone review, then read each applicable reference
completely before inspecting the change:

- For an initial review, read [initial review](./reviewing-code/initial-review.md).
- For a re-review, read [re-review](./reviewing-code/re-review.md). Also read and apply
  [initial review](./reviewing-code/initial-review.md) to every new, expanded, or wholly uncovered
  surface identified by the re-review procedure.
- For a pull request review, read [pull requests](./reviewing-code/pull-requests.md).
- Read [architectural fitness](./reviewing-code/architectural-fitness.md) when the change affects
  executable behaviour, schemas, public or typed contracts, module boundaries, dependency
  relationships, shared components, business rules, or architecture documentation.

Architectural-fitness guidance may be skipped only for prose, formatting, generated output,
repository metadata, or mechanical work with no architectural choice.

Load other repository guidance selected by the affected paths and concerns, including the
repository's Go and test guidance when those surfaces change.

## Establish context and scope

Establish the requested outcome, rationale, intended behaviour, constraints, and acceptance criteria
from the review request and the governing repository documents. Record the sources that establish
each requirement. Together, these sources establish the review contract.

Request requirements and governing documents are binding review criteria. Implementer notes may add
non-conflicting constraints or explain justified supporting work, but cannot narrow the request.
Do not treat cleanup, refactoring, or supporting work outside an explicit requirement as a finding
without a concrete adverse consequence.

An unqualified initial review covers the entire active changeset. For local or ad hoc work, that is
the working tree relative to `HEAD`, including staged, unstaged, and untracked files. For a pull
request, it is the pull-request head relative to its target-branch merge base. An explicit pair of
comparison references supplied by the user overrides those defaults. Use repository state only to
establish the surface; do not report staging, tracking, branch, or commit hygiene as findings.

Discover a local surface with read-only status and diff inspection. Take the union of tracked
changes against `HEAD` and every untracked file, accounting for additions, modifications, deletions,
and renames. Record `HEAD`, any explicit comparison references, and the complete file set under
`Context and sources` so the coverage claim is reproducible.

If the user explicitly narrows the review, record the included and excluded surfaces and claim
completeness only for the declared scope. A review is incomplete until every file and materially
affected flow in that scope has been examined. When the baseline or comparison endpoint cannot be
established, record the ambiguity and use `Review incomplete`.

## Review against the current standard

Judge new and materially changed work against current repository guidance and configuration.
Neighbouring legacy code explains constraints but is not proof that a pattern remains acceptable.
A narrow change need not repair unrelated debt, although copied, extracted, substantially rewritten,
or newly exposed code must meet the current standard.

Trace real control and data flows before reporting a defect. Review correctness, architecture,
security, compatibility, failure handling, recovery, side effects, cross-platform behaviour, and
test effectiveness where relevant. Tests must assert material behaviour rather than merely execute
code. Do not require production tests for prose, configuration, generated output, or mechanical work
when the appropriate repository checks provide the evidence.

Report only observations introduced, worsened, newly exposed, or made directly relevant by the
change. Each finding must state:

- the concrete observation and location;
- the reachable condition, execution path, or governing evidence;
- the user, data, security, operational, architectural, or maintenance consequence; and
- one bounded recommended outcome.

Findings are objective and unranked. Do not add severity, priority, or delivery-impact labels.
Group occurrences with the same root cause and identify every affected location. Do not omit a
supported finding because another one already determines the likely human decision.

Do not report unrelated debt, personal preference, deterministic failures already fully represented
by a recorded repository check, hypothetical risks without a credible path, or redesigns whose only
benefit is aesthetic consistency. Do not manufacture a finding when the change is acceptable.

## Verify each changed surface

Read the root contribution guide's canonical [Verification](../CONTRIBUTING.md#verification) section
and select its required checks for every changed surface. Run only non-mutating review commands and
record each command and result. In particular:

- verify production-code behaviour through stable interfaces at the test level appropriate to the
  change, together with the applicable code-quality, security, build, and end-to-end gates;
- verify documentation with content, link, render, or documentation-specific checks;
- verify configuration with its parser, schema, or tool-native validation;
- verify mechanical edits with searches, diffs, and relevant static checks; and
- verify generated output through its owning generation or staleness check without editing it.

Mixed changes require the combined checks selected for their surfaces. Do not run production
software tests merely to validate prose, configuration, generated output, or a mechanical edit.
Remote CI may provide additional evidence, but it does not replace required local evidence. Fixing,
formatting, snapshot-update, and generation commands are for implementers; reviewers record the
failure and do not mutate the working tree.

When a required command cannot run, record the attempted command, the exact constraint or failure,
and the resulting evidence gap. `Review incomplete` takes precedence over the other verdicts while
any material evidence gap remains. A failed check whose output fully establishes a concrete finding
does not by itself make the review incomplete; record the finding and use `Changes recommended`.

## Write `code-review.md`

Use these sections in this order:

1. **Context and sources** - requested outcome and rationale, review type, declared scope,
   changed surfaces, and every requirement or guidance source read.
2. **Requirements** - complete, explicit acceptance criteria, each attributed to the request or a
   named repository document and section.
3. **Advisory verdict** - exactly one of `No changes recommended`, `Changes recommended`, or
   `Review incomplete`, followed by a concise evidence-based rationale.
4. **Findings** - objective, unranked findings. Write `No findings` when there are none.
5. **Validation evidence** - every surface-appropriate local command with its result, plus targeted
   inspection or behavioural evidence used by the review.
6. **Unverified evidence and questions** - material evidence gaps and decisions that require human
   input. Write `None` when there are none.

Keep each finding actionable and self-contained. The review is complete when the artifact accounts
for every in-scope file and affected flow, every applicable requirement, all prior findings in a
re-review, and all required surface-appropriate verification.
