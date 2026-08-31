# Tests are deterministic

Test outcomes do not depend on timing luck, ordering accidents, external state, or flaky infrastructure.

Deterministic tests support trust in automation. Flaky tests train teams to ignore signals and make reviewers less willing to use the suite as evidence. In review, this is about whether the codebase contains durable evidence for the behaviors it claims to protect. Strong signals are tests that map to meaningful invariants, cover important failure modes, and remain stable under refactoring. Weak signals are flaky tests, over-mocking, unreadable setup, or evidence that only checks implementation trivia rather than user-visible promises. The educational point is that tests are part of the design because they define what the system intends never to break. For this specific symptom, the reviewer should ask whether the change makes 'Tests are deterministic' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
test "order expires after its deadline"
  order = expiringOrder(liveClock, sharedEvents)
  sleep(1 second)
  assert sharedEvents.last == OrderExpired(order.id)
```

### Good

```text
test "order expires after its deadline"
  clock = controlledClock(at: deadline - 1 second)
  events = recordedEvents()
  order = expiringOrder(clock, events)
  clock.advance(2 seconds)
  await events.received(OrderExpired(order.id))
  assert events.all == [OrderExpired(order.id)]
```
