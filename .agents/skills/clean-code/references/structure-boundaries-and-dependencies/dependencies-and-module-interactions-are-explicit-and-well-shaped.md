# Dependencies and module interactions are explicit and well-shaped

Collaborators, dependency direction, coupling, cohesion, and interface shape work together so modules depend on the right things, for the right reasons, through narrow and comprehensible contracts.

Maintainable modular design is not created by any one property in isolation. Explicit dependencies, sane dependency direction, low coupling, high cohesion, and narrow interfaces reinforce each other and should be reviewed as one connected lens. A change can look clean locally while still damaging maintainability if it hides collaborators, lets high-level policy depend on low-level detail, spreads a responsibility across many places, or exposes a broad interface that gives other modules too much reach.

Strong signs include collaborators that are visible in constructors, parameters, or clearly declared module boundaries; dependencies that point inward toward stable policy rather than outward toward incidental framework or storage detail; modules whose internal elements change for related reasons; and interfaces that expose only the capabilities another module actually needs. Weak signs include service locator style lookups, broad context objects passed everywhere, reach-through access into another module's internals, helpers that know too much about distant subsystems, and interfaces that exist only to mirror implementation detail.

The point of this lens is to ask whether the codebase preserves separable responsibilities. When these properties hold together, change stays local, tests are easier to write, and readers can understand why one part of the system depends on another. When they fail, modules become sticky, edits spread unpredictably, and architecture degrades into a mesh of hidden assumptions. A reviewer using this symptom should judge whether the change makes dependencies and module interactions easier to see, safer to change, and harder to misuse.

## Examples

### Bad

```text
function ship(order):
  customer = App.services.database.tables.customers.find(order.customerId)
  transport = App.services.email.transport
  transport.send(customer.email, "Your order shipped")
```

### Good

```text
module Shipping:
  interface ShipmentNotifier: shipped(customerId)
  function ship(order, notifier: ShipmentNotifier):
    notifier.shipped(order.customerId)
module Email:
  class EmailShipmentNotifier implements Shipping.ShipmentNotifier
```
