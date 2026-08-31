# Integration points are exercised realistically

At least some tests cover real wiring between modules, adapters, and system boundaries.

Integration tests prove that individually sensible parts still work together when wiring, serialization, configuration, and infrastructure boundaries are involved. They catch a different class of defect than unit tests. In review, this is about whether the codebase contains durable evidence for the behaviors it claims to protect. Strong signals are tests that map to meaningful invariants, cover important failure modes, and remain stable under refactoring. Weak signals are flaky tests, over-mocking, unreadable setup, or evidence that only checks implementation trivia rather than user-visible promises. The educational point is that tests are part of the design because they define what the system intends never to break. For this specific symptom, the reviewer should ask whether the change makes 'Integration points are exercised realistically' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
test "checkout charges the cart":
  checkout = mock(Checkout)
  payments = mock(PaymentAdapter)
  when checkout.place(cart) => Receipt("paid")
  assert checkout.place(cart).status == "paid"
```

### Good

```text
test "checkout charges the cart":
  gateway = stubServer(returning = 200)
  payments = HttpPaymentAdapter(gateway.url)
  checkout = Checkout(payments)
  receipt = checkout.place(cart)
  assert receipt.status == "paid"
  assert gateway.receivedPaymentFor(cart.total)
```
