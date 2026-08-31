# Code is easy to follow locally

Control flow and structure are linear and bounded enough that a reviewer can understand a unit without chasing large amounts of external context.

Maintainable code should allow mostly local reasoning. A reader should be able to step through a function or unit in a forward pass, with limited jumping across helpers, conditions, and distant files just to understand the main path. The point is not that every piece of context disappears, but that the amount of hidden context stays small enough for the unit to remain intelligible on its own.

Strong signs include control flow that unfolds in a readable sequence; helpers that clarify rather than fragment the story; a unit whose purpose, assumptions, and main branches can be understood without loading half the codebase; and boundaries that keep relevant state and rules nearby. Weak signs include frequent mental backtracking, helper chains that obscure rather than clarify, behavior that only makes sense after discovering hidden state elsewhere, and units whose meaning depends on tribal knowledge or many distant conventions.

This symptom matters because local reasoning is one of the foundations of safe maintenance. If the reader cannot understand the unit in a bounded scope, review quality drops, debugging slows down, and small changes become risky. Code that reads linearly and can be reasoned about locally is easier to test, easier to refactor, and less likely to produce accidental breakage.

## Examples

### Bad

```text
function canShip(orderId):
  order = loadOrder(orderId)
  return passesPolicy(order)
function passesPolicy(order):
  if isBlocked(order): return false
  return hasStock(order)
function hasStock(order):
  return inventory.contains(order.items) and shippingEnabled
```

### Good

```text
function canShip(order, inventory, shippingEnabled):
  if not shippingEnabled: return false
  if order.isBlocked: return false
  if not inventory.contains(order.items): return false
  return true
```
