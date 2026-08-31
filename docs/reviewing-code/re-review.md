# Re-review

A re-review verifies previous findings and reviews code added or changed to address them. It must
not restart a general review of unchanged code or introduce new non-blockers from the original diff.
Start with a compact status: which previous blockers are resolved, which remain, and whether a fix
introduced a regression. Do not make the implementer reconstruct that state from earlier comments.

The initial review is expected to be exhaustive. When every blocker is correctly resolved, every
non-blocker is resolved or consciously deferred, and the remediation introduces no regression,
re-review should produce no new findings from the original diff. A finding first raised against
unchanged original code during re-review is a miss in the initial review, not an ordinary additional
review round.

A regression introduced by a fix is a new finding. A genuinely missed blocker in the original diff
must still be reported, identified candidly as missed in the initial review, and grouped with any
related issue. Do not suppress a material risk merely to preserve review efficiency.
