# Initial review

Establish the required outcome, observable acceptance conditions, explicit constraints, changed
surfaces, and applicable guidance before judging implementation choices.

Focus review effort on judgments automation cannot make: whether the change achieves the required
outcome and follows explicit requirements; whether ticket-silent implementation choices and
additional changes are technically sound and coherent; whether its control and data flows are
correct; whether its boundaries, responsibilities, and dependencies fit the architecture; whether
permissions, failure handling, recovery, compatibility, and side effects are safe; and whether
tests assert the material behaviour rather than merely passing.

Map the high-risk boundaries across the whole diff before writing findings:

- writes, transactions, destructive actions, and partial failure;
- error handling and failures converted into empty or successful states;
- external or model-generated input;
- authentication, authorisation, audit context, and tenant isolation;
- asynchronous work, races, and stale state;
- public, persisted, and integration compatibility; and
- business rules or sources of truth that might be duplicated.

Trace analogous call sites before reporting a defect. Review every changed file and every materially
affected control or data flow, even after finding enough blockers to determine the verdict. Do not
stop at a numeric limit, severity threshold, representative sample, or selection of the most
important findings.

Before publishing, make a final completeness sweep against the required outcome, explicit
requirements, review context, whole diff, affected call sites, tests, applicable guidance, and
high-risk boundary map. Group occurrences with the same root cause in one finding and identify every
affected location. Communicate every supported blocker and non-blocker in one initial review,
ordered by consequence. Priority governs order, not inclusion; do not drip-feed findings as
separate parts of the diff are inspected.
