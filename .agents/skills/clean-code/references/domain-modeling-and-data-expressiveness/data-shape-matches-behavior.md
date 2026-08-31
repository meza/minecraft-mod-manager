# Data shape matches behavior

Data structures reflect access patterns and invariants instead of being generic containers that invite misuse.

Data structures should fit the questions the code asks and the rules the data must obey. Poorly chosen shapes force callers to remember layout conventions or bolt rule checks on later. In review, this is about whether the solution has more moving parts than the requirement has earned. Strong signals are a small number of concepts, one obvious route through the logic, and abstractions that remove repeated cost instead of adding ceremony. Weak signals are speculative generalization, many special cases, flag-driven behavior, and repeated domain rules hiding inside primitive data. The educational point is that unnecessary complexity compounds maintenance cost and makes every later bug harder to isolate. For this specific symptom, the reviewer should ask whether the change makes 'Data shape matches behavior' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
activeMemberIds = List()
function activate(id):
  if not activeMemberIds.contains(id):
    activeMemberIds.add(id)
function isActive(id):
  return activeMemberIds.contains(id)
```

### Good

```text
activeMemberIds = Set()
function activate(id):
  activeMemberIds.add(id)
function isActive(id):
  return activeMemberIds.contains(id)
```
