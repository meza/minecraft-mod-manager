# Framework does not own the design

The architecture is not merely a mirror of the chosen framework's defaults or constraints.

A framework should support the architecture, not define it accidentally. Good code resists letting directory shape, annotations, or framework idioms become the only organizing principle of the system. In review, this is about changeability at module, boundary, and architectural level rather than only about local cleanliness. Strong signals are explicit dependencies, narrow interfaces, coherent module ownership, and boundaries that reflect real responsibilities in the system. Weak signals are reach-through access, policy mixed with mechanism, accidental framework-driven structure, and changes that would force broad unrelated edits. The educational point is that design quality shows up most clearly when requirements evolve or when one part of the system fails. For this specific symptom, the reviewer should ask whether the change makes 'Framework does not own the design' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
framework_model OrderRecord:
  before_save:
    total = subtotal
    if customer_tier == "premium":
      total = subtotal * 0.9
```

### Good

```text
domain Order:
  function totalFor(customerTier):
    if customerTier == "premium": return subtotal * 0.9
    return subtotal
framework_adapter OrderRecord:
  before_save:
    total = Order(subtotal).totalFor(customer_tier)
```
