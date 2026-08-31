# Sensitive data is protected in telemetry

Logs, traces, and messages do not leak secrets, private data, or unsafe internal detail.

Telemetry is part of the system's attack and privacy surface. Review should check that logs, traces, and errors do not expose credentials, personal data, or other sensitive internal detail. In review, this is about whether the running system will be diagnosable and truthful once it leaves the developer's machine. Strong signals are structured telemetry, useful identifiers, protected sensitive data, and runtime signals that reflect the system's real ability to do work. Weak signals are prose-only logging, noisy or incomplete diagnostics, superficial health checks, and missing visibility on critical paths. The educational point is that maintainability includes the ability to understand production behavior, not just source code structure. For this specific symptom, the reviewer should ask whether the change makes 'Sensitive data is protected in telemetry' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
function recordAuthenticationFailure(request):
    telemetry.log("authentication failed", {
        email: request.email,
        password: request.password,
        accessToken: request.headers.authorization
    })
```

### Good

```text
function recordAuthenticationFailure(request):
    telemetry.log("authentication failed", {
        outcome: "denied",
        reason: "invalid_credentials",
        email: "[REDACTED]"
    })
```
