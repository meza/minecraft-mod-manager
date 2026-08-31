# Mutation has a clear owner

When state changes, the owner, lifecycle, and allowed transitions are obvious.

Mutable state is easier to trust when ownership and lifecycle are explicit. A reviewer should be able to tell who is allowed to change it, when it changes, and what states are valid. In review, this is about how much interpretation work the reader must do before they can trust what the code is trying to accomplish. Strong signals are names, structure, and local flow that let another engineer build a correct mental model quickly. Weak signals are vague labels, mixed levels of abstraction, hidden assumptions, or a need to chase many references before the unit makes sense. The educational point is that readability is not cosmetic; it is what makes future change, debugging, and review safe. For this specific symptom, the reviewer should ask whether the change makes 'Mutation has a clear owner' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
order.status = "pending"
warehouse -> order.status = "shipped"
support -> order.status = "cancelled"
billing -> order.status = "refunded"
```

### Good

```text
class Order
  private status = "pending"
  function transitionTo(nextStatus)
    require allowed(status, nextStatus)
    status = nextStatus
warehouse -> order.transitionTo("shipped")
support -> order.transitionTo("cancelled")
billing -> order.transitionTo("refunded")
```
