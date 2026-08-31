# Diagnostics preserve signal

Errors, traces, and metrics contain enough information to diagnose failures without drowning operators in clutter.

Diagnostics should preserve the information needed to understand failures, especially causal context, identifiers, and meaningful error boundaries. Too little signal and too much noise are both costly. In review, this is about whether the running system will be diagnosable and truthful once it leaves the developer's machine. Strong signals are structured telemetry, useful identifiers, protected sensitive data, and runtime signals that reflect the system's real ability to do work. Weak signals are prose-only logging, noisy or incomplete diagnostics, superficial health checks, and missing visibility on critical paths. The educational point is that maintainability includes the ability to understand production behavior, not just source code structure. For this specific symptom, the reviewer should ask whether the change makes 'Diagnostics preserve signal' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
INFO "retrying card payment"
WARN "payment failed"
INFO "retrying card payment"
WARN "payment failed"
INFO "retrying card payment"
ERROR "payment failed"
```

### Good

```text
ERROR event="payment.authorization.failed" dependency="card_gateway"
      trace_id="4bf92f3577b34da6a3ce929d0e0e4736" payment_id="pay_example_4821"
      outcome="retry_exhausted" attempts=3 timeout_ms=2000 elapsed_ms=6420
      error_type="GatewayTimeout" recovery="check_gateway_latency"
```
