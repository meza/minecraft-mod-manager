# Side effects are visible

A reader can tell whether a function mutates state, performs I/O, logs, throws, retries, or causes other external effects.

Good code makes side effects legible. The caller and reviewer should not have to discover by surprise that a function mutates shared state, performs network I O, logs, retries, or triggers other externally visible behavior. In review, this is about how much interpretation work the reader must do before they can trust what the code is trying to accomplish. Strong signals are names, structure, and local flow that let another engineer build a correct mental model quickly. Weak signals are vague labels, mixed levels of abstraction, hidden assumptions, or a need to chase many references before the unit makes sense. The educational point is that readability is not cosmetic; it is what makes future change, debugging, and review safe. For this specific symptom, the reviewer should ask whether the change makes 'Side effects are visible' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
function orderTotal(order):
  rate = exchangeRates.fetch(order.currency)
  order.total = sum(order.items) * rate
  return order.total
```

### Good

```text
function refreshOrderTotal(order, exchangeRates) -> Result<Money, RateFailure>:
  rate = exchangeRates.fetch(order.currency)
  if rate.failed: return Failure(rate.error)
  total = sum(order.items) * rate.value
  order.total = total
  return Success(total)
```
