# Important invariants are tested

Each important rule, guarantee, or failure mode has at least one test that would fail if the guarantee broke.

Important guarantees deserve direct evidence. The stronger the promise, the more the codebase should contain a test or check that would visibly fail if that promise stopped being true. In review, this is about whether the codebase contains durable evidence for the behaviors it claims to protect. Strong signals are tests that map to meaningful invariants, cover important failure modes, and remain stable under refactoring. Weak signals are flaky tests, over-mocking, unreadable setup, or evidence that only checks implementation trivia rather than user-visible promises. The educational point is that tests are part of the design because they define what the system intends never to break. For this specific symptom, the reviewer should ask whether the change makes 'Important invariants are tested' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
test "ten items receive a bulk discount":
  assert checkout(orderWith(10 items)).discountRate == 10%
```

### Good

```text
test "bulk discount starts at ten items":
  assert checkout(orderWith(9 items)).discountRate == 0%
  assert checkout(orderWith(10 items)).discountRate == 10%
```
