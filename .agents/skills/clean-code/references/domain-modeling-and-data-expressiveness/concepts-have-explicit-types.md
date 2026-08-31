# Concepts have explicit types

Important domain concepts are represented explicitly rather than being flattened into vague primitives, strings, flags, and maps.

When a concept matters to correctness, readability, or change safety, it usually deserves an explicit type or structured representation. This makes invariants visible, reduces misuse, and keeps domain rules from leaking into scattered primitive checks. In review, this is about how much interpretation work the reader must do before they can trust what the code is trying to accomplish. Strong signals are names, structure, and local flow that let another engineer build a correct mental model quickly. Weak signals are vague labels, mixed levels of abstraction, hidden assumptions, or a need to chase many references before the unit makes sense. The educational point is that readability is not cosmetic; it is what makes future change, debugging, and review safe. For this specific symptom, the reviewer should ask whether the change makes 'Concepts have explicit types' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
booking = {"attendees": 0}
function reserve(data):
  if data["attendees"] <= 0: error("invalid")
  save(data)
```

### Good

```text
type AttendeeCount private(value)
  create(value):
    require value > 0
    return AttendeeCount(value)
type Booking(attendees: AttendeeCount)
function reserve(booking: Booking): save(booking)
```
