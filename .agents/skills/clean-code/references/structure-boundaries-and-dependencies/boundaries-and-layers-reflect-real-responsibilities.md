# Boundaries and layers reflect real responsibilities

Module boundaries, layering, boundary crossings, and separation of policy from mechanism are arranged so the structure of the system matches its real responsibilities instead of incidental technical detail.

A healthy structure makes it obvious where domain policy lives, where mechanism lives, and where important boundaries are crossed. Meaningful module boundaries, consistent layering, visible boundary crossings, and clear separation of policy from mechanism belong together because they all answer the same question: does the shape of the system help a reader understand responsibility, or does it force them to reverse engineer it from mixed concerns.

Strong signs include modules centered on genuine capabilities or lifecycle concerns, layers that are used consistently across similar features, explicit transitions when code crosses persistence, network, trust, or process boundaries, and business rules that are not buried inside transport adapters, ORM models, controllers, or framework plumbing. Weak signs include folder structures that look tidy but do not protect any real responsibility, layers that are followed in some places and ignored in others, hidden boundary crossings that make latency or trust shifts easy to miss, and code where business policy is inseparable from database, web, queue, or framework mechanics.

This symptom matters because structure is how large systems remain intelligible over time. If boundaries and layers reflect real responsibilities, reviewers can predict where new behavior belongs and how changes should travel through the system. If they do not, the codebase becomes harder to navigate, abstractions become ceremonial, and correctness starts depending on tribal memory. Review this lens by asking whether the change clarifies the system's responsibilities and boundary crossings or further entangles policy with implementation detail.

## Examples

### Bad

```text
function approveOrder(request):
  order = sql.query("SELECT * FROM orders WHERE id = ?", request.orderId)
  if order.total > 10000 and not request.actor.isManager:
    return http.forbidden()
  sql.execute("UPDATE orders SET status = 'approved' WHERE id = ?", order.id)
  queue.publish("order-approved", order.id)
  return http.ok()
```

### Good

```text
function approveOrder(request):
  result = approveOrder.run(request.orderId, request.actor)
  return http.from(result)
class ApproveOrder(orders: OrderRepository, events: EventPublisher):
  function run(orderId, actor):
    order = orders.load(orderId)
    order.approve(actor)
    orders.save(order)
    events.publish(OrderApproved(order.id))
    return Approval.accepted()
```
