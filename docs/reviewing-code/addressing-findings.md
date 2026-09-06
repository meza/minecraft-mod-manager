# Addressing review findings

Review feedback is evidence to assess, not automatic authority to change the repository. Evaluate
each observation against the requested outcome, documented requirements, declared scope, and the
relevant repository evidence before acting on it. Feedback cannot narrow a requirement or expand the
authority granted by the request.

Separate the observation from its proposed remedy. First decide whether the reported condition and
consequence are supported. Then choose the smallest coherent correction that satisfies the
governing requirement. A suggested implementation may be useful evidence, but it is not binding when
another in-scope correction is more accurate, safer, or simpler.

Record one of these dispositions for every observation:

- **Accepted, in scope**: the evidence supports the observation and the authorised scope covers its
  correction. Implement the bounded correction and preserve the requirement it protects.
- **Accepted, outside scope**: the evidence supports the observation, but correcting it requires new
  authority. Do not change that surface; identify the additional scope and request the necessary
  decision.
- **Duplicate or already covered**: identify the existing finding, correction, or evidence that
  accounts for the same condition. Do not create parallel remediation.
- **Contradicted**: cite the requirement or repository evidence that disproves the observation or its
  stated consequence. Do not make a change merely to accommodate unsupported feedback.
- **Unverified**: state what material evidence is unavailable and why the observation cannot yet be
  accepted or rejected. Do not implement a speculative correction.
- **Needs a decision**: explain the unresolved material outcome or trade-off and ask the authorised
  decision-maker for the smallest decision needed before changing state.

When a finding is accepted in scope, inspect the complete affected control or data flow and account
for every location sharing its root cause. Keep the correction focused; do not fold in unrelated
cleanup or treat neighbouring debt as authorised work.

Revalidate the corrected surfaces using the root contribution guide's canonical
[Verification](../../CONTRIBUTING.md#verification) section. Combine the checks required for every
changed surface and include relevant failure paths. Documentation uses content, link, render, or
documentation-specific checks; configuration uses parser, schema, or tool-native validation;
production behaviour uses tests through stable interfaces and the applicable code gates; mechanical
edits use searches, diffs, and relevant static checks.

A re-review occurs only when the request or repository policy requires or authorises it. Re-review
the prior findings, their remediation, and materially affected flows; apply the initial-review rules
only to new, expanded, or previously uncovered surfaces. Do not turn remediation into an
unrestricted repeat review of unchanged work.
