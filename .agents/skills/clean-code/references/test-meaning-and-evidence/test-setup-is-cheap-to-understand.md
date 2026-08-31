# Test setup is cheap to understand

Creating the conditions for a test does not require navigating a maze of fixtures, mocks, or magical helpers.

Complicated fixtures and magical helpers obscure the actual behavior under test. Good test setup makes the relevant preconditions easy to see and minimizes incidental ceremony. In review, this is about whether the codebase contains durable evidence for the behaviors it claims to protect. Strong signals are tests that map to meaningful invariants, cover important failure modes, and remain stable under refactoring. Weak signals are flaky tests, over-mocking, unreadable setup, or evidence that only checks implementation trivia rather than user-visible promises. The educational point is that tests are part of the design because they define what the system intends never to break. For this specific symptom, the reviewer should ask whether the change makes 'Test setup is cheap to understand' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
test "suspends account after third failed payment":
  world = magicalFixture("delinquent customer")
  account = world.customer().primaryAccount()
  service = world.container().resolve("payment service")
  result = service.recordFailure(account.id)
  expect result.status = "suspended"
```

### Good

```text
test "suspends account after third failed payment":
  account = AccountBuilder().withFailedPayments(2).build()
  accounts = InMemoryAccounts(account)
  service = PaymentService(accounts)
  result = service.recordFailure(account.id)
  expect result.status = "suspended"
```
