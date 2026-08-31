# Tests encode behavior, not implementation trivia

Tests describe externally meaningful promises rather than internal structure that should be free to change.

Tests should protect behavior users and maintainers actually care about. Over-coupling tests to implementation detail makes refactoring expensive without increasing real confidence. In review, this is about whether the codebase contains durable evidence for the behaviors it claims to protect. Strong signals are tests that map to meaningful invariants, cover important failure modes, and remain stable under refactoring. Weak signals are flaky tests, over-mocking, unreadable setup, or evidence that only checks implementation trivia rather than user-visible promises. The educational point is that tests are part of the design because they define what the system intends never to break. For this specific symptom, the reviewer should ask whether the change makes 'Tests encode behavior, not implementation trivia' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
test "successful checkout":
  checkout = spyInternals(Checkout())
  checkout.buy(cart)
  assert checkout._validate called with cart
  assert checkout._reserveStock called after checkout._validate
  assert checkout._chargeCard called after checkout._reserveStock
  assert checkout._createReceipt called after checkout._chargeCard
```

### Good

```text
test "successful checkout":
  payments = FakePayments(); inventory = FakeInventory()
  checkout = Checkout(payments, inventory)
  receipt = checkout.buy(cart)
  assert receipt.status == "paid"
  assert receipt.items == cart.items
  assert receipt.total == cart.total
  assert payments.chargeFor(receipt.orderId) == cart.total
  assert inventory.isReserved(receipt.orderId, cart.items)
```
