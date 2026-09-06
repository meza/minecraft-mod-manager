# Pull request reviews

Read the pull request's title, full description, target branch, head revision, and discussion before reviewing the change. Include general comments, earlier reviews, inline threads, replies, and resolved or outdated discussions where available. Check the full discussion, not just the most recent comments.

Note the pull request and revisions reviewed. If relevant history is unavailable, explain what is missing and how it affects the review. Use `Review incomplete` when the gap prevents you from establishing requirements, scope, the comparison, or the state of a finding. If the available evidence is sufficient, explain why the missing history does not affect the conclusion.

Use the description and discussion to understand purpose, boundaries, constraints, earlier findings, implementation decisions, and questions already resolved. They do not override the request's requirements, accepted architecture, or current repository guidance. A different implementation or additional change is not a finding by itself; identify a concrete adverse consequence.

Review the full agreed scope even when a comment highlights a smaller area. For an explicitly limited review, state the boundary and do not claim coverage beyond it.

Run the checks required by the [review guide](../code-review.md#verify-each-changed-surface) and record the commands and results. CI provides additional evidence, but does not replace local verification or determine the verdict by itself. The implementer decides how the advisory review affects the pull request.
