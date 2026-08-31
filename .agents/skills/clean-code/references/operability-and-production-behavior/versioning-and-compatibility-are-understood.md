# Versioning and compatibility are understood

The system handles evolving contracts, clients, data shapes, and deployment skew with intention.

Compatibility is a long-lived design concern in APIs, events, data formats, and deployments. Review should consider how a change behaves when old and new producers or consumers coexist. In review, this is about whether the system can survive evolution, skew, retries, failures, and operational reality without losing coherence. Strong signals are deliberate recovery stories, compatibility-aware boundaries, reproducible delivery, navigable structure, and interfaces that make correct use easier than misuse. Weak signals are brittle migrations, unexamined version skew, hidden build assumptions, and local shortcuts that quietly increase long-term coupling. The educational point is that maintainability is proven over time, especially when the system is under stress or in transition. For this specific symptom, the reviewer should ask whether the change makes 'Versioning and compatibility are understood' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
DEPLOY POST /payments REQUIRES {amount, currency}
old_client -> POST /payments {amount}
server -> 400 missing currency
```

### Good

```text
KEEP POST /payments {amount} -> charge(amount, "GBP")
ADD POST /v2/payments {amount, currency} -> charge(amount, currency)
DEPLOY server supporting legacy + v2 before v2 clients
MIGRATE clients; retire /payments after compatibility_window
```
