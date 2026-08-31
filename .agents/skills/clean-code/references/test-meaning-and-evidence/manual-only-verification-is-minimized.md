# Manual-only verification is minimized

Important correctness claims are not left resting on ad hoc human checking when they could be mechanized.

When verification can be automated, leaving it manual usually lowers repeatability and increases review burden. Manual checking still has a place, but high-value correctness claims should not depend on memory and patience alone. In review, this is about whether the codebase contains durable evidence for the behaviors it claims to protect. Strong signals are tests that map to meaningful invariants, cover important failure modes, and remain stable under refactoring. Weak signals are flaky tests, over-mocking, unreadable setup, or evidence that only checks implementation trivia rather than user-visible promises. The educational point is that tests are part of the design because they define what the system intends never to break. For this specific symptom, the reviewer should ask whether the change makes 'Manual-only verification is minimized' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
release checklist "declined payment keeps the order unpaid":
  submit checkout with the declined test card
  observe the payment error
  open the order record
  observe status "unpaid"
  tick the checklist item
```

### Good

```text
test "declined payment keeps the order unpaid":
  gateway will decline the payment
  order = submit checkout
  expect payment error is shown
  expect order.status equals "unpaid"
release gate runs test on every change
```
