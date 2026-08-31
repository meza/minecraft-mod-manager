# Recovery paths are intentional

Restart, retry, replay, rollback, and repair stories exist where the domain requires them.

Recovery paths are part of maintainability because broken systems must be repairable. Review should ask whether replay, rollback, retry, restart, or manual recovery are possible where the domain needs them. In review, this is about whether the system can survive evolution, skew, retries, failures, and operational reality without losing coherence. Strong signals are deliberate recovery stories, compatibility-aware boundaries, reproducible delivery, navigable structure, and interfaces that make correct use easier than misuse. Weak signals are brittle migrations, unexamined version skew, hidden build assumptions, and local shortcuts that quietly increase long-term coupling. The educational point is that maintainability is proven over time, especially when the system is under stress or in transition. For this specific symptom, the reviewer should ask whether the change makes 'Recovery paths are intentional' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
payment.charge(order.total)
while true:
  try:
    inventory.reserve(order.items)
    return CONFIRMED
  catch failure:
    continue
```

### Good

```text
payment.chargeOnce(order.id, order.total)
for attempt in 1..3:
  try: inventory.reserveOnce(order.id, order.items); break
  catch failure: if failure.transient: wait(backoff(attempt)); else: break
state = inventory.reservation(order.id)
if state == RESERVED: inventory.confirmOnce(order.id); return CONFIRMED
cancelled = inventory.cancelOnce(order.id)
refunded = payment.refundOnce(order.id, order.total)
if cancelled and refunded: return FAILED
markRecoveryPending(order.id); return RECOVERY_PENDING
```
