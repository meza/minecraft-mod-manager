# Regression evidence exists

Changed behavior is backed by tests or checks that prove the intended effect and protect against relapse.

A system learns from failure when regressions become tests, checks, or other durable evidence. That creates a memory of past mistakes inside the codebase rather than inside individual people. In review, this is about whether the codebase contains durable evidence for the behaviors it claims to protect. Strong signals are tests that map to meaningful invariants, cover important failure modes, and remain stable under refactoring. Weak signals are flaky tests, over-mocking, unreadable setup, or evidence that only checks implementation trivia rather than user-visible promises. The educational point is that tests are part of the design because they define what the system intends never to break. For this specific symptom, the reviewer should ask whether the change makes 'Regression evidence exists' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
bug: padded port value crashes
fix parse_port(value):
  return integer(trim(value))
ship fix
```

### Good

```text
test padded_port_is_accepted:
  assert parse_port(" 443 ") == 443
run test => FAIL
fix parse_port(value):
  return integer(trim(value))
run test => PASS
```
