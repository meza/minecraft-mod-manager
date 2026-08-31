# Re-review

A re-review is bounded to the prior matching review, the remediation, and any flows materially
affected by that remediation. It does not restart an unrestricted review of unchanged original
work.

Use the existing root `code-review.md` only when both the work item and review surface match. A
matching ticket identifier establishes work-item identity but does not establish surface identity.
For ad hoc work, require the same rationale. In either case, reconcile the current file and flow set
with the complete file set recorded in the previous artifact. Apply bounded re-review to files that
were in that prior set and to flows already covered there. Give every new or renamed file, newly
deleted file, and flow expanded beyond the prior review an initial review by applying
[initial-review guidance](./initial-review.md), and record that split. Files returned to the baseline
need no further review. When none of the current surface was covered previously, apply the complete
initial-review procedure.

For each prior finding, record whether it is resolved, remains, or cannot be verified. Carry an
unresolved finding forward with enough original context to remain actionable, and identify it as a
repeated finding. Inspect every file and affected flow changed to address the prior review, and
report any regression introduced by the remediation.

Do not introduce a new finding about unchanged original work merely to broaden the review. If a
previously missed issue presents a concrete material risk, report it candidly as missed during the
initial review; review efficiency does not justify suppressing material evidence.

Run all required local gates again and record their current results. Select the advisory verdict
from the evidence now available. Use `Review incomplete` whenever a material evidence gap remains;
otherwise use `Changes recommended` when findings remain and `No changes recommended` when none do.
