# Nondeterminism is controlled

Time, randomness, concurrency, and network variability are explicit and testable rather than leaking unpredictably into logic.

Time, randomness, scheduling, and remote behavior introduce uncertainty. Good design isolates these influences behind explicit seams so logic remains understandable, testable, and less flaky. In review, this is about whether the code preserves truth under bad input, surprising state, and real failure conditions rather than only under ideal conditions. Strong signals are explicit contracts, visible failure behavior, and boundaries that stop invalid or unsafe state from spreading. Weak signals are silent fallback, ambiguous error handling, implicit ordering requirements, and cleanup that only works on the happy path. The educational point is that correctness depends as much on what happens when things go wrong as on what happens when they go right. For this specific symptom, the reviewer should ask whether the change makes 'Nondeterminism is controlled' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
function issueTrial():
  code = randomString(8)
  expiresAt = currentTime() + 7 days
  return Trial(code, expiresAt)
```

### Good

```text
function issueTrial(clock, random):
  code = random.string(8)
  expiresAt = clock.now() + 7 days
  return Trial(code, expiresAt)
```
