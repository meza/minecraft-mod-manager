# Deletion is easy

Dead code, outdated paths, and obsolete abstractions can be removed without fear of hidden dependency mines.

Deletion is a sign of design health. If obsolete code can be removed with confidence, the boundaries and dependencies are probably visible; if not, hidden coupling is likely too high. In review, this is about changeability at module, boundary, and architectural level rather than only about local cleanliness. Strong signals are explicit dependencies, narrow interfaces, coherent module ownership, and boundaries that reflect real responsibilities in the system. Weak signals are reach-through access, policy mixed with mechanism, accidental framework-driven structure, and changes that would force broad unrelated edits. The educational point is that design quality shows up most clearly when requirements evolve or when one part of the system fails. For this specific symptom, the reviewer should ask whether the change makes 'Deletion is easy' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
globalFlags["gift_wrap"] = true
checkout.total += giftWrapFee(cart)
receipt.fields["gift_wrap"] = cart.giftWrap
analytics.emit("gift_wrap", receipt.fields["gift_wrap"])
supportQueue.routeOn("gift_wrap", receipt)
delete giftWrapFee  // the flag, field, event, and route remain
```

### Good

```text
giftWrap = GiftWrapCapability(pricing, analytics, supportQueue)
checkout = Checkout(capabilities = [giftWrap])
GiftWrapCapability owns quote, receipt, reporting, and routing
delete GiftWrapCapability
checkout = Checkout(capabilities = [])
```
