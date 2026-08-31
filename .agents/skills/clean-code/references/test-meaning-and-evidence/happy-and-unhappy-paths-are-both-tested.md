# Happy and unhappy paths are both tested

Negative cases, edge cases, and misuse paths are intentionally exercised rather than left implicit.

Most failures do not happen on the ideal path. Good test suites make invalid input, edge cases, dependency errors, and misuse behavior explicit rather than assuming the happy path tells the whole story. In review, this is about whether the codebase contains durable evidence for the behaviors it claims to protect. Strong signals are tests that map to meaningful invariants, cover important failure modes, and remain stable under refactoring. Weak signals are flaky tests, over-mocking, unreadable setup, or evidence that only checks implementation trivia rather than user-visible promises. The educational point is that tests are part of the design because they define what the system intends never to break. For this specific symptom, the reviewer should ask whether the change makes 'Happy and unhappy paths are both tested' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
test "registers a new account":
  accounts = empty accounts
  expect register(accounts, "alex@example.test") = Registered
```

### Good

```text
test "registers a new account":
  accounts = empty accounts
  expect register(accounts, "alex@example.test") = Registered
test "rejects an email already in use":
  accounts = accounts containing "alex@example.test"
  expect register(accounts, "alex@example.test") = EmailAlreadyUsed
  expect accounts.count = 1
```
