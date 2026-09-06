# Re-review

A re-review is bounded to the prior matching review, the remediation, and any flows materially
affected by that remediation. It does not restart an unrestricted review of unchanged original
work.

Use the existing root `code-review.md` only when its requested outcome, constraints, acceptance
criteria, and review surface match the current request. Reconcile the current file and flow set with
the complete file set recorded in the previous artifact. Apply bounded re-review to files that were
in that prior set and to flows already covered there. Give every new or renamed file, newly deleted
file, and flow expanded beyond the prior review an initial review by applying [initial-review
guidance](./initial-review.md), and record that split. Files returned to the baseline need no further
review. When none of the current surface was covered previously, apply the complete initial-review
procedure.

For each prior finding, record whether it is resolved, remains, or cannot be verified. Carry an
unresolved finding forward with enough original context to remain actionable, and identify it as a
repeated finding. Inspect every file and affected flow changed to address the prior review, and
report any regression introduced by the remediation.

Do not introduce a new finding about unchanged original work merely to broaden the review. If a
previously missed issue presents a concrete material risk, report it candidly as missed during the
initial review; review efficiency does not justify suppressing material evidence.

Repeat the surface-appropriate verification required by the canonical review policy and record the
current results. Select the advisory verdict from the evidence now available. Use `Review incomplete`
whenever a material evidence gap remains; otherwise use `Changes recommended` when findings remain
and `No changes recommended` when none do.
