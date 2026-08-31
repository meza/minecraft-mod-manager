# Tests are readable

A reviewer can tell what behavior a test protects and why it matters.

Readable tests act as executable documentation for behavior. A reviewer should be able to see what scenario is being protected, what outcome matters, and why that outcome is significant. In review, this is about whether the codebase contains durable evidence for the behaviors it claims to protect. Strong signals are tests that map to meaningful invariants, cover important failure modes, and remain stable under refactoring. Weak signals are flaky tests, over-mocking, unreadable setup, or evidence that only checks implementation trivia rather than user-visible promises. The educational point is that tests are part of the design because they define what the system intends never to break. For this specific symptom, the reviewer should ask whether the change makes 'Tests are readable' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
test "p":
  x = mk(50, false, 0)
  y = svc(stub(2))
  z = y.do(x)
  assert z.s == 1
  assert x.c == 0
```

### Good

```text
test "declined payment leaves order awaiting payment":
  // Arrange
  order = unpaidOrder(total: 50)
  checkout = checkoutWith(paymentResult: Declined)
  // Act
  checkout.pay(order)
  // Assert
  assert order.status == AwaitingPayment
  assert order.amountCharged == 0
```
