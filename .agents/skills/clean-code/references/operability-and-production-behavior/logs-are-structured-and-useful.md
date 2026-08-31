# Logs are structured and useful

Production logs carry useful context and machine-readable shape instead of noise or free-form guessing text.

Useful logs capture enough structured context to support diagnosis and correlation without forcing operators to parse prose. They should help answer what happened, where, and under which identifiers or conditions. In review, this is about whether the running system will be diagnosable and truthful once it leaves the developer's machine. Strong signals are structured telemetry, useful identifiers, protected sensitive data, and runtime signals that reflect the system's real ability to do work. Weak signals are prose-only logging, noisy or incomplete diagnostics, superficial health checks, and missing visibility on critical paths. The educational point is that maintainability includes the ability to understand production behavior, not just source code structure. For this specific symptom, the reviewer should ask whether the change makes 'Logs are structured and useful' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
log("Starting payment")
log("Processing order " + order.id)
log("Payment failed: " + error.message)
log("Done")
```

### Good

```text
log.event("payment.failed", {
  order_id: order.id, payment_provider: provider.name,
  error_code: error.code, trace_id: trace.id
})
```
