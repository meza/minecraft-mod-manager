# Related things live together

Logic, data, and rules that belong together are not scattered across distant files or layers.

High maintainability depends on proximity of related behavior, rules, and data. When logic that changes together is split across many places, readers lose context and changes start missing one of the required touch points. In review, this is about how much interpretation work the reader must do before they can trust what the code is trying to accomplish. Strong signals are names, structure, and local flow that let another engineer build a correct mental model quickly. Weak signals are vague labels, mixed levels of abstraction, hidden assumptions, or a need to chase many references before the unit makes sense. The educational point is that readability is not cosmetic; it is what makes future change, debugging, and review safe. For this specific symptom, the reviewer should ask whether the change makes 'Related things live together' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
config/shipping:
  STANDARD_FEE = 5
checkout/order_totals:
  shipping = 0 if subtotal >= 50 else config.STANDARD_FEE
receipts/shipping_label:
  label = "Free" if order.shipping == 0 else "Standard"
```

### Good

```text
module ShippingPolicy:
  FREE_OVER = 50
  STANDARD = ShippingOption(label: "Standard", fee: 5)
  FREE = ShippingOption(label: "Free", fee: 0)
  quote(subtotal) = FREE if subtotal >= FREE_OVER else STANDARD
Checkout.shipping = ShippingPolicy.quote(subtotal)
Receipt.shippingLabel = order.shipping.label
```
