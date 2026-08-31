# Public contracts are documented

Public or easy-to-misuse interfaces explain their guarantees, constraints, error cases, side effects, and expected usage.

Some interfaces can be misused even when their code is clean. For those, documentation should explain constraints, error modes, side effects, ordering assumptions, and any subtle contract points that are not obvious from the type signature alone. In review, this is about how much interpretation work the reader must do before they can trust what the code is trying to accomplish. Strong signals are names, structure, and local flow that let another engineer build a correct mental model quickly. Weak signals are vague labels, mixed levels of abstraction, hidden assumptions, or a need to chase many references before the unit makes sense. The educational point is that readability is not cosmetic; it is what makes future change, debugging, and review safe. For this specific symptom, the reviewer should ask whether the change makes 'Public contracts are documented' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
// Schedules a transfer.
public scheduleTransfer(accountId, amount, requestId) -> Transfer
```

### Good

```text
// Queues a transfer for asynchronous execution; use the returned transfer's status to track it.
// amount must be positive and use the account's currency.
// Reusing requestId returns the original transfer without queuing another debit.
// Returns a PENDING transfer and emits TransferQueued after the request is persisted.
// Throws AccountNotFound or InvalidAmount; a failed call has no side effects.
public scheduleTransfer(accountId, amount, requestId) -> Transfer
```
