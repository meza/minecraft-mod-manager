# Unsafe defaults are avoided

The easiest path through the code does not accidentally produce insecure, destructive, or privacy-harming behavior.

Defaults shape the behavior most users and maintainers will get. Safe defaults make the easiest path the least dangerous one, especially for destructive, public, or security-sensitive features. In review, this is about how the code treats trust, privilege, secrets, and untrusted input. Strong signals are explicit trust boundaries, least privilege, safe defaults, and data handling that is minimal and auditable. Weak signals are embedded credentials, casual exposure of sensitive data, weak validation, and code paths whose security depends on convention rather than structure. The educational point is that secure design is part of normal code quality because unsafe code is inherently harder to change and reason about. For this specific symptom, the reviewer should ask whether the change makes 'Unsafe defaults are avoided' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
function createReportLink(report, publicAccess = true):
    if publicAccess:
        return linkStore.publish(report)
    return linkStore.restrict(report, currentUser)
```

### Good

```text
function createReportLink(report, publicAccess = false):
    if not publicAccess:
        return linkStore.restrict(report, currentUser)
    require currentUser.confirmed("publish report publicly")
    return linkStore.publish(report)
```
