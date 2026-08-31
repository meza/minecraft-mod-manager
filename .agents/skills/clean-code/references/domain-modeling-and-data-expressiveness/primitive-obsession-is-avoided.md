# Primitive obsession is avoided

Repeated low-level primitives do not stand in for richer domain concepts that deserve first-class representation.

Primitive obsession keeps domain meaning out of the type system and inside scattered convention. Replacing raw strings, maps, and numbers with domain types makes intent clearer and invariants easier to enforce. In review, this is about whether the solution has more moving parts than the requirement has earned. Strong signals are a small number of concepts, one obvious route through the logic, and abstractions that remove repeated cost instead of adding ceremony. Weak signals are speculative generalization, many special cases, flag-driven behavior, and repeated domain rules hiding inside primitive data. The educational point is that unnecessary complexity compounds maintenance cost and makes every later bug harder to isolate. For this specific symptom, the reviewer should ask whether the change makes 'Primitive obsession is avoided' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
function shipOrder(order, street, city, postalCode):
  if blank(street) or blank(city) or not validPostalCode(postalCode): fail
  order.shippingStreet = street; order.shippingCity = city; order.shippingPostalCode = postalCode
function estimateDelivery(street, city, postalCode):
  if blank(street) or blank(city) or not validPostalCode(postalCode): fail
  return carrier.quote(street, city, postalCode)
```

### Good

```text
value object ShippingAddress:
  create(street, city, postalCode):
    require not blank(street) and not blank(city) and validPostalCode(postalCode)
    return ShippingAddress(street, city, postalCode)
function shipOrder(order, address: ShippingAddress):
  order.shippingAddress = address
function estimateDelivery(address: ShippingAddress):
  return carrier.quote(address)
```
