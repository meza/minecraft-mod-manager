# Health signals reflect real health

Health endpoints and readiness checks reflect whether the service can actually perform its role.

A health signal is only valuable if it reflects real ability to do useful work. Superficial liveness checks can create false confidence while critical dependencies are broken. In review, this is about whether the running system will be diagnosable and truthful once it leaves the developer's machine. Strong signals are structured telemetry, useful identifiers, protected sensitive data, and runtime signals that reflect the system's real ability to do work. Weak signals are prose-only logging, noisy or incomplete diagnostics, superficial health checks, and missing visibility on critical paths. The educational point is that maintainability includes the ability to understand production behavior, not just source code structure. For this specific symptom, the reviewer should ask whether the change makes 'Health signals reflect real health' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
order-service /health:
  return 200 "alive"
```

### Good

```text
order-service /live:
  return 200 "alive"
order-service /ready:
  if database.ping(timeout) and orderQueue.canPublish(timeout):
    return 200 "ready"
  return 503 "not ready"
```
