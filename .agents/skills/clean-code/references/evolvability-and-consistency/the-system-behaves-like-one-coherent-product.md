# The system behaves like one coherent product

From function semantics up to architecture, the whole system expresses a unified set of rules, patterns, and expectations rather than conflicting local micro-cultures.

Coherence means the system feels like one thing built according to one set of principles. From local functions to top-level architecture, patterns, error behavior, naming, and boundaries reinforce one another instead of competing. In review, this is about whether the system can survive evolution, skew, retries, failures, and operational reality without losing coherence. Strong signals are deliberate recovery stories, compatibility-aware boundaries, reproducible delivery, navigable structure, and interfaces that make correct use easier than misuse. Weak signals are brittle migrations, unexamined version skew, hidden build assumptions, and local shortcuts that quietly increase long-term coupling. The educational point is that maintainability is proven over time, especially when the system is under stress or in transition. For this specific symptom, the reviewer should ask whether the change makes 'The system behaves like one coherent product' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
ProductOrderRules.isOpen(order) = order.status in [PENDING, PROCESSING]
billing.openOrders = orders.filter(order.status != SHIPPED)
support.openOrders = orders.filter(order.status == PENDING)
```

### Good

```text
ProductOrderRules.isOpen(order) = order.status in [PENDING, PROCESSING]
billing.openOrders = orders.filter(ProductOrderRules.isOpen)
support.openOrders = orders.filter(ProductOrderRules.isOpen)
```
