# Error semantics are intentional and consistent

Failures are surfaced in deliberate, diagnosable, and system-consistent ways, without silent semantic drift, swallowed exceptions, or arbitrary local error styles.

Error handling is part of the contract of a system, not cleanup logic added after the happy path. Good code makes it clear what can fail, how failure is represented, what degradation is acceptable, and what must be surfaced immediately. It also uses the same broad error vocabulary and handling style across similar situations so that callers and operators do not have to relearn failure semantics for each module.

Strong signs include explicit failure modes, consistent representation of similar errors, propagation or transformation that preserves meaning, and fallback behavior that either preserves semantics or clearly signals degradation. Weak signs include catch-all blocks that erase useful information, silent fallback paths that change what the system means, modules that mix exceptions, return codes, logs, and null-like sentinels arbitrarily, and error handling that leaves the system in an unclear state.

This symptom matters because many serious defects are not caused by the initial failure but by confusion about what the failure meant. Inconsistent or hidden error semantics make systems difficult to diagnose, difficult to integrate with safely, and easy to corrupt under stress. A reviewer should be able to see not only that failures are handled, but that they are handled truthfully and in line with the rest of the system.

## Examples

### Bad

```text
function loadOrder(id):
  try: return orders.find(id)
  catch NotFound: return null
  catch error:
    log("load failed")
    return null
```

### Good

```text
function loadOrder(id) -> Result<Order, OrderFailure>:
  try: return Ok(orders.find(id))
  catch NotFound as cause:
    return Err(OrderFailure.NotFound(orderId: id, cause: cause))
  catch Timeout or ConnectionFailure as cause:
    return Err(OrderFailure.Unavailable(orderId: id, cause: cause))
```
