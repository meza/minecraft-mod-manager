# Function shape is cohesive and tractable

A function has one clear job, stays small enough to understand as a whole, and takes a set of inputs that reflects its true responsibility.

A good function is cohesive in purpose, tractable in size, and honest about what it depends on. These properties reinforce each other. When a function truly has one job, its flow tends to stay compact enough to hold in working memory, and its parameters tend to reflect the minimum meaningful inputs for that responsibility rather than a grab bag of context.

Strong signs include a function whose name describes one responsibility, whose internal steps all serve that responsibility, whose size does not force the reader through multiple unrelated conceptual phases, and whose parameters reveal the data or collaborators the function genuinely needs. Weak signs include orchestration, validation, transformation, persistence, logging, and formatting all mixed together; long functions that scroll through several mental modes; parameter lists that are bloated, generic, or obviously shaped around caller convenience; and context objects passed only because the function has unclear boundaries.

This symptom matters because function shape is where maintainability is often won or lost. Overgrown or unfocused functions become hard to review, hard to test, and hard to change without collateral damage. A cohesive and bounded function makes intent visible, isolates reasons to change, and gives reviewers a realistic chance of understanding the whole unit rather than merely skimming fragments of it.

## Examples

### Bad

```text
function processOrder(order, pricing, invoices, mailer, logger):
  require order.hasItems()
  total = pricing.totalFor(order.items)
  invoice = Invoice(order.customer, total)
  invoices.save(invoice)
  mailer.send(invoice)
  logger.record("invoice sent")
```

### Good

```text
function fulfillOrder(order, pricing, invoices, mailer, logger):
  invoice = createInvoice(order, pricing)
  invoices.save(invoice)
  mailer.send(invoice)
  logger.record("invoice sent")

function createInvoice(order, pricing):
  require order.hasItems()
  return Invoice(order.customer, pricing.totalFor(order.items))
```
