# Reviewers do not need tribal memory

Correctness does not depend heavily on undocumented norms that only long-time contributors know.

Tribal knowledge is a hidden dependency. Good code and supporting artifacts expose the rules that matter so correctness does not depend on who happens to review the change. In review, this is about whether the codebase behaves like one system instead of a collection of local habits. Strong signals are repeated solutions to repeated problems, stable naming and error semantics, and cross-cutting concerns that follow shared rules. Weak signals are multiple local styles for the same problem, contradictory artifacts, and correctness that depends on knowing tribal lore. The educational point is that consistency reduces cognitive load and makes local review findings more trustworthy. For this specific symptom, the reviewer should ask whether the change makes 'Reviewers do not need tribal memory' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
function deploy(change):
  // Release captains know which services need two approvals.
  production.deploy(change)
```

### Good

```text
policy = DeploymentPolicy.fromRepository("deployment-policy.yml")
function deploy(change, policy):
  policy.requireApprovals(change)
  production.deploy(change)
ci.requires("deployment-policy-check")
```
