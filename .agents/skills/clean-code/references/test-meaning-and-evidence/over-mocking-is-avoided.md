# Over-mocking is avoided

Tests do not become brittle by asserting every internal interaction instead of meaningful outcomes.

Mocking should isolate external concerns without turning tests into scripts of internal implementation steps. Over-mocking makes refactors painful and can create false confidence in brittle designs. In review, this is about whether the codebase contains durable evidence for the behaviors it claims to protect. Strong signals are tests that map to meaningful invariants, cover important failure modes, and remain stable under refactoring. Weak signals are flaky tests, over-mocking, unreadable setup, or evidence that only checks implementation trivia rather than user-visible promises. The educational point is that tests are part of the design because they define what the system intends never to break. For this specific symptom, the reviewer should ask whether the change makes 'Over-mocking is avoided' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
test "checkout completes payment":
  pricing = mock(); gateway = mock(); receipts = mock()
  expect pricing.total(cart) -> 50
  expect gateway.authorize(card, 50) -> "auth-7"
  expect gateway.capture("auth-7")
  expect receipts.create(cart, 50) -> receipt
  checkout = Checkout(pricing, gateway, receipts)
  checkout.complete(cart, card)
  verify_in_order(pricing, gateway, receipts)
```

### Good

```text
test "checkout completes payment":
  gateway = FakePaymentGateway()
  checkout = Checkout(Pricing(), gateway, Receipts())
  receipt = checkout.complete(cart_with(item(price: 50)), card)
  assert_equal(receipt.status, "paid")
  assert_equal(receipt.total, 50)
  assert_equal(gateway.charges, [charge(card, 50)])
```
