# Security boundaries are explicit

Trust levels, privilege transitions, and sensitive operations are visible in the design and code paths.

Security boundaries separate trust zones, privileges, and sensitive operations. Maintainable secure code makes these boundaries explicit so reviewers can see where stronger checks are required. In review, this is about how the code treats trust, privilege, secrets, and untrusted input. Strong signals are explicit trust boundaries, least privilege, safe defaults, and data handling that is minimal and auditable. Weak signals are embedded credentials, casual exposure of sensitive data, weak validation, and code paths whose security depends on convention rather than structure. The educational point is that secure design is part of normal code quality because unsafe code is inherently harder to change and reason about. For this specific symptom, the reviewer should ask whether the change makes 'Security boundaries are explicit' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
function deleteAccount(request):
    privilegedAccounts.delete(request.accountId)
```

### Good

```text
function deleteAccount(request, identity):
    authorizationBoundary.require(identity, "account.delete", request.accountId)
    privilegedAccounts.delete(request.accountId)
```
