# Initial review

Establish the work item's required outcome, observable acceptance conditions, explicit constraints,
changed surfaces, and applicable guidance before judging implementation choices.

Use the comparison endpoints defined by the canonical review policy. An unqualified review covers
every new, modified, deleted, or renamed file in that active changeset and every materially affected
control or data flow. If the user explicitly narrows the review, record the boundary in
`code-review.md`, inspect every file and flow inside it, list excluded surfaces, and do not claim
completeness outside it.

Map high-risk boundaries across the declared surface before writing findings:

- writes, destructive actions, partial failure, and recovery;
- errors converted into empty or successful states;
- external, untrusted, or generated input;
- authentication, authorisation, audit context, and tenant isolation;
- asynchronous work, races, retries, and stale state;
- public, persisted, and integration compatibility; and
- business rules or sources of truth that may have been duplicated.

Focus on judgments automation cannot make: whether the outcome and constraints are satisfied;
whether control and data flows are correct; whether boundaries, responsibilities, and dependencies
fit the target architecture; whether permissions, failure handling, compatibility, and side effects
are safe; and whether tests assert the material behaviour.

Trace analogous call sites before concluding that an implementation is defective. Continue after
finding an issue: do not stop at a numeric limit, a consequence threshold, a representative sample,
or the first evidence supporting `Changes recommended`.

Before publishing, reconcile the work item, declared scope, every in-scope file, affected call sites
and flows, tests, applicable guidance, high-risk boundary map, and required local-gate evidence.
Group a shared root cause into one finding and identify every affected location. The first review
must contain every supported finding so remediation can be completed in one coherent pass.
