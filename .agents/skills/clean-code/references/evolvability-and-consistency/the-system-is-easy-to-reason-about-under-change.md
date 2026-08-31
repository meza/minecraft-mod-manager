# The system is easy to reason about under change

When requirements evolve, it is obvious where to look, what to protect, and what else might be affected.

Maintainability is tested most when requirements change. Good structure helps reviewers infer where else a change may matter and which invariants are likely to be at risk. In review, this is about whether the system can survive evolution, skew, retries, failures, and operational reality without losing coherence. Strong signals are deliberate recovery stories, compatibility-aware boundaries, reproducible delivery, navigable structure, and interfaces that make correct use easier than misuse. Weak signals are brittle migrations, unexamined version skew, hidden build assumptions, and local shortcuts that quietly increase long-term coupling. The educational point is that maintainability is proven over time, especially when the system is under stress or in transition. For this specific symptom, the reviewer should ask whether the change makes 'The system is easy to reason about under change' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
checkout: shipping = total >= 50 ? 0 : 5
cartBanner: show "Free shipping over 50"
receipt: label = total >= 50 ? "Free" : "£5"
change threshold: edit checkout, cartBanner, receipt, and their tests
```

### Good

```text
policy = ShippingPolicy(freeAt = 50, standardFee = 5)
checkout = Checkout(shippingPolicy = policy)
cartBanner = CartBanner(shippingPolicy = policy)
receipt = Receipt(shippingPolicy = policy)
change threshold: edit ShippingPolicy and its contract tests
```
