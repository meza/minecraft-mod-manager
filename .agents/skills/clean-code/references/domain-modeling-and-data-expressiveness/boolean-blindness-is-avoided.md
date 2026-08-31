# Boolean blindness is avoided

Flags and ambiguous booleans are not used where richer types or named concepts would express intent better.

Boolean blindness happens when a bare true or false carries too much unstated meaning. Named concepts, enums, richer types, or separate functions often communicate intent and valid combinations better. In review, this is about whether the solution has more moving parts than the requirement has earned. Strong signals are a small number of concepts, one obvious route through the logic, and abstractions that remove repeated cost instead of adding ceremony. Weak signals are speculative generalization, many special cases, flag-driven behavior, and repeated domain rules hiding inside primitive data. The educational point is that unnecessary complexity compounds maintenance cost and makes every later bug harder to isolate. For this specific symptom, the reviewer should ask whether the change makes 'Boolean blindness is avoided' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
scheduleDelivery(order, true)
```

### Good

```text
type DeliverySpeed = Standard | Expedited
scheduleDelivery(order, DeliverySpeed.Expedited)
```
