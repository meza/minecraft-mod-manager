# Secrets are not embedded

Credentials, tokens, keys, and sensitive configuration are not hard-coded into source or artifacts.

Secrets in source tend to spread into history, logs, screenshots, and local clones. Good systems externalize secrets and handle them through safer configuration and secret-management mechanisms. In review, this is about how the code treats trust, privilege, secrets, and untrusted input. Strong signals are explicit trust boundaries, least privilege, safe defaults, and data handling that is minimal and auditable. Weak signals are embedded credentials, casual exposure of sensitive data, weak validation, and code paths whose security depends on convention rather than structure. The educational point is that secure design is part of normal code quality because unsafe code is inherently harder to change and reason about. For this specific symptom, the reviewer should ask whether the change makes 'Secrets are not embedded' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
function charge(card, amount):
    apiKey = "hardcoded-production-secret"
    return payments.charge(apiKey, card, amount)
```

### Good

```text
function charge(card, amount, secretStore):
    apiKey = secretStore.read("payments/api-key")
    return payments.charge(apiKey, card, amount)
```
