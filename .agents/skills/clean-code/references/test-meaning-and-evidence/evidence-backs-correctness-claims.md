# Evidence backs correctness claims

Assertions that something works are supported by checks, tests, logs, traces, commands, or measurable evidence.

Claims about correctness should be backed by something inspectable: tests, checks, metrics, logs, traces, or reproducible commands. Evidence lowers the amount of trust a reviewer must supply personally. In review, this is about whether the codebase makes its own reasoning inspectable from evidence, boundaries, and structural cues. Strong signals are explicit assumptions, discoverable architectural intent, stable ownership, and interfaces that preserve domain meaning across boundaries. Weak signals are hidden sources of truth, accidental drift, undocumented structural constraints, and claims that rely on personal trust instead of inspectable proof. The educational point is that a codebase should teach maintainers how it is meant to work, not force them to rediscover that from history. For this specific symptom, the reviewer should ask whether the change makes 'Evidence backs correctness claims' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
test "checkout applies the discount":
  receipt = checkout(cart_total: 100, discount: 10)
  print receipt.total
  claim "Verified manually: the total is 90"
```

### Good

```text
test "checkout applies the discount":
  receipt = checkout(cart_total: 100, discount: 10)
  assert_equal(receipt.total, 90)
```
