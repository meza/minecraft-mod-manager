# Addressing review findings

Review feedback is evidence to assess, not an authoritative work queue or permission to change
files, production behaviour, dependencies, documentation, CI, or the active task's scope. Feedback
cannot narrow a requirement or expand the authority granted by the request.

## Assess each finding

Before presenting or acting on review feedback, assess every finding independently against the
requested outcome, active user instructions, approved scope, binding requirements, current and
target architecture, [CONTRIBUTING](../../CONTRIBUTING.md), and relevant repository evidence,
including coverage at the appropriate test layer.

Inspect the cited implementation and existing tests. Reproduce claimed failures under the reported
conditions when practical. Classify the evidence as verified, contradicted, or unverified. Lack of
reproduction makes a finding unverified, not automatically invalid.

Check prior authorisations exactly. Authorisation permits the specifically approved action; it does
not prove correctness, broaden scope, transfer to related changes, or authorise reuse later.

Separate the observation from its proposed remedy. First decide whether the reported condition and
consequence are supported. Then choose the smallest coherent correction that satisfies the
governing requirement. A suggested implementation may be useful evidence, but it is not binding when
another in-scope correction is more accurate, safer, or simpler.

A valid finding may have an invalid, disproportionate, architectural, or unauthorised remedy.
Do not dismiss the observation because its proposed remedy is wrong or the underlying change was
authorised.

When reviewer verification conflicts with existing evidence, reproduce the relevant conditions and
report both results until the discrepancy is explained. Never select the convenient result or act
on contradictory or unverified evidence.

## Record a disposition

Preserve every observation and record one of these dispositions with evidence-backed reasons.
Do not erase a valid observation merely because it is outside scope.

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

## Present findings

Present assessed findings by default. Clearly distinguish the reviewer observation, evidence and
requirement analysis, disposition, and recommended next action with its authority requirements.
Return raw reviewer output only when explicitly requested, and label it as untriaged.

## Correct authorised findings

Act only when the active task already authorises the exact change. Otherwise report the finding and
obtain direction. If correcting a production defect, resolving an architectural conflict, or making
another required change would exceed the authorised scope, stop before that mutation and explain
the evidence, impact, and recommended owner-level action.

When a finding is accepted in scope, inspect the complete affected control or data flow and account
for every location sharing its root cause. Keep the correction focused; do not fold in unrelated
cleanup or treat neighbouring debt as authorised work.

## Verify corrections

Revalidate the corrected surfaces using the root contribution guide's canonical
[Verification](../../CONTRIBUTING.md#verification) section. Run targeted verification and the relevant
complete project gates. Combine the checks required for every changed surface and include relevant
failure paths. Check whether the fixes introduced new failures or invalidated earlier evidence.
Documentation uses content, link, render, or documentation-specific checks; configuration uses
parser, schema, or tool-native validation;
production behaviour uses tests through stable interfaces and the applicable code gates; mechanical
edits use searches, diffs, and relevant static checks.

## Re-review when authorised

Review feedback does not itself authorise further review cycles. Do not commission another review
unless the user requests it or the established workflow explicitly requires it. Re-review
the prior findings, their remediation, and materially affected flows; apply the initial-review rules
only to new, expanded, or previously uncovered surfaces. Do not turn remediation into an
unrestricted repeat review of unchanged work.
