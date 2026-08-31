# Return values are meaningful

Outputs communicate useful meaning rather than ambiguous booleans, magic values, or overloaded null semantics.

Return values should encode useful outcomes and failure states clearly. They should not force callers to remember sentinel values, overloaded null meanings, or fragile positional conventions. In review, this is about how much interpretation work the reader must do before they can trust what the code is trying to accomplish. Strong signals are names, structure, and local flow that let another engineer build a correct mental model quickly. Weak signals are vague labels, mixed levels of abstraction, hidden assumptions, or a need to chase many references before the unit makes sense. The educational point is that readability is not cosmetic; it is what makes future change, debugging, and review safe. For this specific symptom, the reviewer should ask whether the change makes 'Return values are meaningful' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
function reserveSeat(showId):
  if show is missing: return null
  if show is sold out: return false
  reservation = createReservation()
  if reservation failed: return -1
  return reservation.id
result = reserveSeat(showId)
if result: confirm(result)
else: informCustomer()
```

### Good

```text
type ReservationResult = Reserved(reservation) | SoldOut | ReservationFailed(reason)
function reserveSeat(showId) -> ReservationResult:
  if show is missing: return ReservationFailed("show not found")
  if show is sold out: return SoldOut
  reservation = createReservation() or return ReservationFailed("creation failed")
  return Reserved(reservation)
match reserveSeat(showId):
  Reserved(reservation) -> confirm(reservation)
  SoldOut -> informCustomer()
  ReservationFailed(reason) -> explainFailure(reason)
```
