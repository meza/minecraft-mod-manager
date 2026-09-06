# Re-review

Re-review the findings, their fixes, and affected flows. Apply [initial-review guidance](./initial-review.md) to new or expanded work.

For each prior finding, record whether it is resolved, remains, or cannot be verified. Carry an unresolved finding forward with enough original context to remain actionable, and identify it as a repeated finding. Inspect every file and affected flow changed to address the prior review, and report any regression introduced by the remediation.

Do not introduce a new finding about unchanged original work merely to broaden the review. If a previously missed issue presents a concrete material risk, report it candidly as missed during the initial review; review efficiency does not justify suppressing material evidence.

Repeat the surface-appropriate verification required by the canonical review policy and record the current results. Select the advisory verdict from the evidence now available. Use `Review incomplete` whenever a material evidence gap remains; otherwise use `Changes recommended` when findings remain and `No changes recommended` when none do.
