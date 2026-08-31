# Sensitive actions are auditable

Important security or privacy-relevant events can be traced after the fact.

Auditability matters when actions affect security, money, privacy, or trust. The code should leave enough evidence to reconstruct what happened and who or what triggered it. In review, this is about how the code treats trust, privilege, secrets, and untrusted input. Strong signals are explicit trust boundaries, least privilege, safe defaults, and data handling that is minimal and auditable. Weak signals are embedded credentials, casual exposure of sensitive data, weak validation, and code paths whose security depends on convention rather than structure. The educational point is that secure design is part of normal code quality because unsafe code is inherently harder to change and reason about. For this specific symptom, the reviewer should ask whether the change makes 'Sensitive actions are auditable' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
requirePrivilege(actor, "delete_user")
users.delete(user)
```

### Good

```text
transaction(users, auditEvents):
  outcome = "denied"
  if authorized(actor, "delete_user"):
    outcome = users.tryDelete(user) ? "success" : "failed"
  auditEvents.appendDurably({
    occurred_at: clock.utcNow(), actor: actor.account,
    action: "user.delete", target: user.account, outcome: outcome,
    origin: request.source, reason: request.reason
  })
```
