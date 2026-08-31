# Extension points are deliberate

The system has explicit seams for likely variation instead of forcing invasive modification for every new need.

Extension points should exist where variation is likely and stable enough to justify a seam. They should not require invasive surgery for every new behavior, nor sprawl prematurely across the design. In review, this is about changeability at module, boundary, and architectural level rather than only about local cleanliness. Strong signals are explicit dependencies, narrow interfaces, coherent module ownership, and boundaries that reflect real responsibilities in the system. Weak signals are reach-through access, policy mixed with mechanism, accidental framework-driven structure, and changes that would force broad unrelated edits. The educational point is that design quality shows up most clearly when requirements evolve or when one part of the system fails. For this specific symptom, the reviewer should ask whether the change makes 'Extension points are deliberate' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
interface CheckoutHook:
  run(event, context)
class PostalRateHook implements CheckoutHook
class CourierRateHook implements CheckoutHook
function shippingCost(order, carrier, hooks):
  context = {"parcel": order.parcel, "destination": order.destination}
  hooks[carrier].run("shipping.rate", context)
  return context["shippingCost"]
```

### Good

```text
interface ShippingRate:
  quote(parcel, destination)
class PostalRate implements ShippingRate
class CourierRate implements ShippingRate
function shippingCost(order, rate: ShippingRate):
  return rate.quote(order.parcel, order.destination)
```
