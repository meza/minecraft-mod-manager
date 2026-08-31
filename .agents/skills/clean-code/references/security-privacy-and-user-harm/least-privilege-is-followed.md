# Least privilege is followed

Code, services, and components operate with the minimum permissions they need.

Least privilege limits the damage of mistakes and compromise. Components should receive only the access they need, and reviewers should be able to see that access decisions are deliberate. In review, this is about how the code treats trust, privilege, secrets, and untrusted input. Strong signals are explicit trust boundaries, least privilege, safe defaults, and data handling that is minimal and auditable. Weak signals are embedded credentials, casual exposure of sensitive data, weak validation, and code paths whose security depends on convention rather than structure. The educational point is that secure design is part of normal code quality because unsafe code is inherently harder to change and reason about. For this specific symptom, the reviewer should ask whether the change makes 'Least privilege is followed' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
function sendInvoice(invoice, database: DatabaseAdmin):
  customer = database.readAnyCustomer(invoice.customerId)
  database.updateAnyInvoice(invoice.id, status: "sent")
  mailer.send(customer.email, invoice)
```

### Good

```text
capability InvoiceDelivery for authorizedInvoice:
  customerEmail()
  markSent()
function sendInvoice(delivery: InvoiceDelivery):
  mailer.send(delivery.customerEmail(), delivery.authorizedInvoice)
  delivery.markSent()
```
