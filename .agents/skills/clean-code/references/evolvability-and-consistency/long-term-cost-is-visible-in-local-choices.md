# Long-term cost is visible in local choices

Local shortcuts that create systemic maintenance debt are either rejected or explicitly acknowledged.

Long-term cost awareness asks whether a local shortcut creates future ambiguity, coupling, or drift. Good review does not reject every shortcut, but it should make systemic cost visible and deliberate. In review, this is about whether the system can survive evolution, skew, retries, failures, and operational reality without losing coherence. Strong signals are deliberate recovery stories, compatibility-aware boundaries, reproducible delivery, navigable structure, and interfaces that make correct use easier than misuse. Weak signals are brittle migrations, unexamined version skew, hidden build assumptions, and local shortcuts that quietly increase long-term coupling. The educational point is that maintainability is proven over time, especially when the system is under stress or in transition. For this specific symptom, the reviewer should ask whether the change makes 'Long-term cost is visible in local choices' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
checkout(order):
  rate = order.country == "GB" ? 0.20 : taxService.rate(order)
  return total(order, rate)
```

### Good

```text
taxRates = TaxRates(primary = taxService,
                     override = TemporaryGbRate(0.20, owner = "Tax Platform",
                                                expires = "2026-10-01"))
checkout(order, taxRates):
  return total(order, taxRates.for(order))
```
