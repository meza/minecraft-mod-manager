---
name: addressing-code-review-findings
description: MUST CONSULT when reading or acting on code-review feedback
---

# Addressing Code-Review Findings

Code reviews surface independent observations. They are evidence, not an authoritative work queue, mutation permission, or permission to expand the active task. Reviewers may be correct, mistaken, contentious, or correct about a problem while recommending the wrong remedy.

Before presenting or acting on review feedback, adjudicate every finding independently.

## Adjudication

For each finding:

1. Separate the reported observation from its proposed remedy.
2. Inspect the cited implementation and existing tests. Reproduce claimed failures under the reported conditions when practical.
3. Classify the evidence as verified, contradicted, or unverified. Lack of reproduction makes a finding unverified, not automatically invalid.
4. Compare the observation with active user instructions, approved task scope, binding requirements, current and target architecture, CONTRIBUTING, and coverage at the appropriate test layer.
5. Check prior authorizations exactly. Authorization permits the specifically approved action; it does not prove correctness, broaden scope, transfer to related changes, or authorize reuse later.
6. Assign one disposition:
   - **accepted—in scope**
   - **accepted—out of scope**
   - **already covered or duplicate**
   - **invalid**
   - **unverified**
   - **decision required**
7. Preserve every observation and give evidence-backed reasons for its disposition. Do not erase a valid observation merely because it is out of scope.
8. Evaluate the proposed remedy separately. A valid finding may have an invalid, disproportionate, architectural, or unauthorized remedy.

When reviewer verification conflicts with existing evidence, reproduce the relevant conditions and report both results until the discrepancy is explained. Never select the convenient result.

## Presenting findings

Present adjudicated findings by default. Clearly distinguish:

- reviewer observation;
- evidence and requirement analysis;
- disposition;
- recommended next action and its authority requirements.

Return raw reviewer output only when explicitly requested, and label it as untriaged.

## Acting

Review feedback does not itself authorize file changes, production behavior changes, dependency changes, documentation changes, CI changes, or further review cycles.

Act only when the active task already authorizes the exact change. Otherwise report the finding and obtain direction. If a finding reveals a production defect, architectural conflict, or required change outside the authorized scope, stop before mutation and explain the evidence, impact, and recommended owner-level action.

After fixing accepted findings, run targeted verification and the relevant complete project gates. Check whether the fixes introduced new failures or invalidated earlier evidence. Do not commission another review unless the user requests it or the established workflow explicitly requires it.

## Common failure modes

- Treating reviewer output as an instruction queue.
- Forwarding raw observations as accepted defects.
- Dismissing a concern merely because the underlying change was authorized.
- Dismissing a valid observation because its suggested remedy is wrong.
- Silently dropping valid but out-of-scope findings.
- Expanding the task into unrelated tooling or quality improvements.
- Acting on contradictory or unverified evidence.
- Starting another review-and-fix cycle without authority.
