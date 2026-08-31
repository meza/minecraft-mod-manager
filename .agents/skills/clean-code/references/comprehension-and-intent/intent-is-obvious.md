# Intent is obvious

A reader can understand what this unit is for without reverse engineering the whole surrounding system.

This asks whether the purpose of the unit is apparent from its shape, naming, and placement. A reader should be able to answer what problem it solves, what role it plays, and roughly where to look next without reconstructing hidden intent from implementation detail. In review, this is about how much interpretation work the reader must do before they can trust what the code is trying to accomplish. Strong signals are names, structure, and local flow that let another engineer build a correct mental model quickly. Weak signals are vague labels, mixed levels of abstraction, hidden assumptions, or a need to chase many references before the unit makes sense. The educational point is that readability is not cosmetic; it is what makes future change, debugging, and review safe. For this specific symptom, the reviewer should ask whether the change makes 'Intent is obvious' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
module jobs
class Processor
  function run(items):
    for item in items:
      if item.status == "held" and item.expiresAt < now():
        item.status = "open"
        save(item)
```

### Good

```text
module reservation_expiry
class ExpiredReservationReleaser
  function releaseExpiredReservations(reservations, currentTime):
    for reservation in reservations:
      if reservation.isHeld and reservation.expiresAt < currentTime:
        reservation.release()
        save(reservation)
```
