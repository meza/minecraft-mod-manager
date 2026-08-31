# Observability matches critical paths

The system emits signals around the behaviors that matter most to users and operators.

Observability should follow user and operator pain, not just implementation convenience. Important journeys, failure modes, and bottlenecks should leave visible traces in the system's signals. In review, this is about whether the running system will be diagnosable and truthful once it leaves the developer's machine. Strong signals are structured telemetry, useful identifiers, protected sensitive data, and runtime signals that reflect the system's real ability to do work. Weak signals are prose-only logging, noisy or incomplete diagnostics, superficial health checks, and missing visibility on critical paths. The educational point is that maintainability includes the ability to understand production behavior, not just source code structure. For this specific symptom, the reviewer should ask whether the change makes 'Observability matches critical paths' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
function checkout(cart):
  metrics.observe("worker.memory_bytes", process.memory)
  metrics.observe("cache.entry_count", cache.size)
  return payment.authorize(cart.total)
```

### Good

```text
function checkout(cart):
  span = traces.start("checkout", cart.id)
  started, outcome = clock.now(), "exception"
  try:
    result = payment.authorize(cart.total)
    outcome = result.outcome
    return result
  finally:
    metrics.record("checkout", outcome, clock.now() - started)
    span.finish(outcome=outcome)
```
