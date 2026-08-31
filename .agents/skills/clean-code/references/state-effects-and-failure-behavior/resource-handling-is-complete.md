# Resource handling is complete

Files, sockets, locks, transactions, and other resources are acquired and released correctly on both success and failure paths.

Resource safety covers acquisition, release, cleanup, and exceptional paths. A maintainable system makes it hard to leak file handles, connections, locks, transactions, or partial work when things go wrong. In review, this is about whether the code preserves truth under bad input, surprising state, and real failure conditions rather than only under ideal conditions. Strong signals are explicit contracts, visible failure behavior, and boundaries that stop invalid or unsafe state from spreading. Weak signals are silent fallback, ambiguous error handling, implicit ordering requirements, and cleanup that only works on the happy path. The educational point is that correctness depends as much on what happens when things go wrong as on what happens when they go right. For this specific symptom, the reviewer should ask whether the change makes 'Resource handling is complete' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
file = open("orders.csv")
orders = parse(file)  // failure skips close
save(orders)
file.close()
```

### Good

```text
with file = open("orders.csv"):
    orders = parse(file)
    save(orders)
// scope closes file on success or failure
```
