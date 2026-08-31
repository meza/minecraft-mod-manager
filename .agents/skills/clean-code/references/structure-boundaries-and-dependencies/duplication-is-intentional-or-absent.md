# Duplication is intentional or absent

Repeated logic is either eliminated or clearly justified, rather than copied accidentally across the codebase.

Duplication is costly when the same rule can drift in several places. Some repetition is acceptable if it keeps boundaries clean, but accidental duplication of knowledge is a common source of inconsistent behavior. In review, this is about whether the solution has more moving parts than the requirement has earned. Strong signals are a small number of concepts, one obvious route through the logic, and abstractions that remove repeated cost instead of adding ceremony. Weak signals are speculative generalization, many special cases, flag-driven behavior, and repeated domain rules hiding inside primitive data. The educational point is that unnecessary complexity compounds maintenance cost and makes every later bug harder to isolate. For this specific symptom, the reviewer should ask whether the change makes 'Duplication is intentional or absent' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
BookingApi.canCancel(booking) =
  hoursUntil(booking.start) >= 24
ReminderJob.canCancel(booking) =
  hoursUntil(booking.start) >= 24
```

### Good

```text
CancellationPolicy.canCancel(booking) =
  hoursUntil(booking.start) >= 24
BookingApi.canCancel(booking) =
  CancellationPolicy.canCancel(booking)
ReminderJob.canCancel(booking) =
  CancellationPolicy.canCancel(booking)
```
