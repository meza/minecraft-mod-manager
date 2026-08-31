# Levels of abstraction are aligned

A unit of code operates at one conceptual level at a time instead of mixing policy, orchestration, formatting, and low-level detail.

A unit should operate at a coherent conceptual altitude. Mixing business policy with formatting, transport, persistence, or framework plumbing forces the reader to reason across multiple layers at once and makes future extraction harder. In review, this is about how much interpretation work the reader must do before they can trust what the code is trying to accomplish. Strong signals are names, structure, and local flow that let another engineer build a correct mental model quickly. Weak signals are vague labels, mixed levels of abstraction, hidden assumptions, or a need to chase many references before the unit makes sense. The educational point is that readability is not cosmetic; it is what makes future change, debugging, and review safe. For this specific symptom, the reviewer should ask whether the change makes 'Levels of abstraction are aligned' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
function notify_overdue(invoice)
  if invoice.days_overdue > 30
    subject = "Payment overdue: " + invoice.number
    body = render_template("collection", invoice.customer)
    smtp = connect("mail.internal", port = 587)
    smtp.send(invoice.customer.email, subject, body)
```

### Good

```text
function notify_overdue(invoice)
  if not requires_collection_notice(invoice)
    return
  notice = compose_collection_notice(invoice)
  send_notice(invoice.customer, notice)
```
