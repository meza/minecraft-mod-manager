# Code review

Code review protects delivery. It prevents a change from adding bugs, spreading legacy problems, or
creating inconsistencies that make the system harder to evolve. It must do that without turning a
focused pull request into a redesign or an endless sequence of review rounds.

Deadlines do not justify regressions. They do affect where valid improvement work belongs. Fix
material risks in the current change; handle useful but deferrable improvements pragmatically and
keep delivery moving.

## Load the guidance for this review

Read this document completely for every code review, then read each applicable reference completely
before inspecting the change:

- For an initial review, read [initial review](./reviewing-code/initial-review.md).
- For a re-review, read [re-review](./reviewing-code/re-review.md) instead of the initial-review
  guidance.
- For a pull request review, read [pull requests](./reviewing-code/pull-requests.md).
- Read [architectural fitness](./reviewing-code/architectural-fitness.md) when the change affects
  executable behaviour, schemas, public or typed contracts, module or workspace boundaries,
  dependency relationships, shared components, business rules, or architecture documentation.

The architecture reference may be skipped only when the change affects none of those surfaces and
is limited to prose, formatting, generated output, repository metadata, or mechanical work with no
architectural choice.

## Review against the right standard

Judge new and materially changed code against the current documented guidance and repository
configuration. Neighbouring legacy code explains the constraints around a change, but it is not
evidence that a pattern remains acceptable.

A narrow change does not require unrelated legacy cleanup. It may use a poor legacy interface when
replacing that interface is outside scope, provided the interaction is contained and does not make
the problem harder to remove. Code that is substantially rewritten, copied, extracted, moved, given
new responsibilities, or exposed through a new interface must meet the current standard.

A ticket defines the required delivery outcome and explicit constraints, not every permissible
change in the contribution. Cleanup, refactoring, supporting work, and additional changes do not
become findings merely because the ticket did not mention them. Assess them on their engineering
merits and report only a concrete correctness, architecture, safety, maintainability, release,
rollback, or review-coherence consequence.

New code must remain understandable, appropriately typed, testable, limited in responsibility,
and explicit about side effects. Review unsafe typing, hidden state, broad exception handling or
suppressions, and dead or debug code according to their concrete consequence. Block them when they
create a material risk; report localised debt as non-blocking when the change remains safe and
functional.

Current documented guidance owns the standard. When guidance conflicts, remains ambiguous, or is
still evolving, do not invent a rule during review. Raise the decision as non-blocking follow-up
unless the implementation already has a concrete correctness or safety defect.

## Use validation evidence

Run targeted validation only when existing evidence does not cover a material behaviour or when a
concrete concern cannot be resolved from that evidence. Compare a failure with the merge-base
baseline when one exists so unrelated failures are not attributed to the change. The
[contribution guide](../CONTRIBUTING.md#required-local-checks) owns validation command selection.

## Classify findings by consequence

Report only findings introduced, worsened, newly exposed, or made directly relevant by the change.
Every finding must have evidence, a credible failure or maintenance consequence, and an actionable
outcome. A policy's force does not determine severity; the consequence does.

### Blockers must be resolved here

A blocker prevents approval because the change introduces or worsens a credible risk of:

- incorrect required behaviour, unmet explicit acceptance conditions, or violation of an explicit
  constraint or prohibition;
- data loss, corruption, partial updates, inconsistent results, or invalid state;
- missing authentication or authorisation, injection, unsafe commands or paths, secret or
  sensitive-data exposure, unsafe input or deserialisation, or cross-tenant access;
- a broken primary workflow or material availability failure;
- incompatible public, integration, or persisted behaviour;
- unsafe deployment, migration, or operational behaviour;
- a material architectural inconsistency that would spread through new consumers; or
- meaningful changed behaviour having no effective regression protection when a stable test seam
  exists.

Trace the real control and data flow before blocking. Describe the input, state, or sequence that
reaches the defect and the user, data, security, or operational consequence. Do not rely on a name,
a test's existence, or a speculative possibility as proof.

Tests must verify behaviour rather than merely execute code. Require success, failure, boundary,
state-transition, validation, permission, or regression coverage only where it protects behaviour
material to the change. When preserving untested legacy behaviour matters, require a
characterisation test at a stable seam. Do not require tests for formatting, generated output, or a
trivial mechanical change already covered by repository checks.

**Tautological tests considered harmful.**

Comments are an explicit surface where a policy's force determines severity: a comment introduced
or modified by the change that violates the
[comment policy](./contributing/code-comments.md)
is a blocker regardless of consequence. The required outcome is deletion or reduction to a
compliant comment, and that fix does not restart validation.

Cross-platform tooling is also an explicit blocker. Trace the tooling's filesystem, process,
environment, and shell boundaries. New or changed shared tooling that cannot run natively on
supported Windows, Linux, and macOS violates the
[cross-platform tooling contract](./contributing/cross-platform-tooling.md) and must be corrected
before approval. Portability is a review concern, not a reason to require operating-system-specific
or matrix tests. Tooling is an exception to the general test-coverage blocker above: do not block
approval because tooling lacks tests. Recommend tests only when complex behaviour would benefit
from regression protection.


### Non-blockers are also surfaced

There's no such thing as a non-blocker. A review needs to surface all observations. Whether it's a blocker or not is determined by the team, not the review.

### Some observations are not findings

Do not report:

- unrelated existing debt;
- a ticket-silent implementation choice or additional change without a concrete adverse
  consequence;
- personal style or naming preferences;
- formatting, lint, or other deterministic failures already reported by repository checks;
- hypothetical failure modes without a credible execution path;
- broad redesigns that would only be nicer; or
- additional test cases with no meaningful behavioural or regression value.

Do not manufacture a finding when the change is acceptable.

## Give one actionable review

Make the next action obvious and keep the context needed to take it close to the finding.

The top-level or banner comment is a decision surface, not an essay. Its first line states the
immediate action or outcome. Do not open with process narration, generic praise, or a recap of the
change. Do not close with pleasantries or an invitation to continue the discussion.

Keep each finding to one bounded action. Lead with a short title that names the defect, then give
only the trigger or failure path, consequence, evidence, and required outcome needed to act. Use
literal, matter-of-fact language. Keep uncertainty only when it changes the decision. Omit tangents,
repeated context, and speculative advice.

Do not repeat full location-specific findings in the banner. Group them by root cause and
consequence, and put detailed evidence at the narrowest useful location. If there are many findings,
preserve all supported findings but split them into blockers and non-blockers rather than producing
one long, unranked list. Concision governs presentation, not completeness.

Every review contains these sections, omitting an empty findings section rather than inventing
content:

1. **Verdict** - approve or request changes.
2. **Blockers** - the trigger, failure path, consequence, evidence, and required outcome.
3. **Non-blockers** - the consequence, whether to consider resolving it now, and a bounded
   follow-up ticket recommendation when deferring it.
4. **Validation** - the checks and behaviour examined, including anything material that could not
   be verified.

An acceptable change receives approval. A review is complete when the implementer can make one
coherent pass over all required fixes, consciously accept or defer every non-blocker, and return for
a bounded verification rather than another excavation of the original change.
