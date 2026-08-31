# The system is resilient to partial failure

Dependency failure, timeout, bad input, and degraded subsystems do not cause disproportionate collapse.

Partial failure resilience means the system can degrade, retry, isolate, or recover without turning every dependency issue into a full outage or corrupt state. This is both an architectural and code-level quality concern. In review, this is about whether the system can survive evolution, skew, retries, failures, and operational reality without losing coherence. Strong signals are deliberate recovery stories, compatibility-aware boundaries, reproducible delivery, navigable structure, and interfaces that make correct use easier than misuse. Weak signals are brittle migrations, unexamined version skew, hidden build assumptions, and local shortcuts that quietly increase long-term coupling. The educational point is that maintainability is proven over time, especially when the system is under stress or in transition. For this specific symptom, the reviewer should ask whether the change makes 'The system is resilient to partial failure' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
function productPage(id):
  product = catalogue.get(id)
  suggestions = recommendations.get(id)
  // A stalled dependency fills the shared worker pool.
  return render(product, suggestions)
```

### Good

```text
function productPage(id):
  product = catalogue.get(id)
  degraded = false
  try:
    suggestions = isolatedRecommendations.get(id, timeout: 200ms)
  catch Timeout:
    suggestions = []
    degraded = true
  return render(product, suggestions, degraded)
```
