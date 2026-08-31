# Pull request reviews

Read the current pull request title and complete body, then read the complete existing discussion
history before establishing the review baseline. This includes general comments, submitted review
bodies, inline review threads, and every reply, including resolved or outdated threads.

Use that context to understand the stated purpose, explicit boundaries, reviewer expectations, known
constraints, prior findings, implementation choices, ticket-silent work, and decisions already
discussed. Treat it as evidence, not independent authority to change the ticket's required outcome
or explicit requirements, or to override accepted architecture and current documented guidance.
A different implementation from one suggested in Jira, or an additional change Jira did not
mention, is not a conflict or finding by itself. Assess it against the canonical review standard and
stop only for an actual conflict between explicit requirements and binding guidance.

Use the repository CI configuration and current-head check list only to establish which validation
CI owns. Do not rerun lint, formatting, typechecking, builds, tests, or any other CI-owned check
locally. CI ownership, not a green result, makes that local verification wasteful: the rule applies
whether a check's current result is green, red, pending, skipped, or absent.

CI status is not a review finding and does not affect the review verdict. Do not report CI results
in the findings, verdict, or validation summary. GitHub's
[required checks and merge controls](../contributing/commits-and-pull-requests.md#satisfy-the-merge-gates)
own enforcement; a failing required check will prevent delivery until the implementer fixes it. Run
targeted validation only for material behaviour CI does not own or when a concrete concern requires
evidence beyond the check's contract.

## Use GitHub's review facilities

When acting as a peer reviewer on a GitHub pull request, collect findings in one pending review and
submit it through GitHub's native review controls. Attach a location-specific finding to the
relevant changed line. Put a cross-cutting finding in the review body, or attach it to the clearest
affected line and identify the other locations there. Do not duplicate the same root cause across
several comments.

Submit **Request changes** when any blocker remains and **Approve** when none remain, including when
the review contains only non-blockers. Do not substitute a general comment for the native decision
or repeat the control label in the review text. A native review state carries the verdict; do not
duplicate it in prose. If the reviewer cannot use the required control, state that limitation in
validation rather than implying that the pull request was approved or blocked.
