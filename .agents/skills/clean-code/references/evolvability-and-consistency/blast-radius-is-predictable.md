# Blast radius is predictable

The likely impact of changing a rule, dependency, schema, or interface is limited and understandable.

Predictable blast radius means contributors can estimate what else may break when a rule, schema, or dependency changes. This supports safer review, testing, and rollout planning. In review, this is about changeability at module, boundary, and architectural level rather than only about local cleanliness. Strong signals are explicit dependencies, narrow interfaces, coherent module ownership, and boundaries that reflect real responsibilities in the system. Weak signals are reach-through access, policy mixed with mechanism, accidental framework-driven structure, and changes that would force broad unrelated edits. The educational point is that design quality shows up most clearly when requirements evolve or when one part of the system fails. For this specific symptom, the reviewer should ask whether the change makes 'Blast radius is predictable' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
Rule: Gold customers receive a 10% discount
Checkout: if tier is GOLD, discount subtotal by 10%
Invoice: if tier is GOLD, show "10% loyalty discount"
Refund: if tier is GOLD, reverse the 10% discount
Change request: increase the Gold discount to 12%
Required edits: Checkout, Invoice, Refund, and their tests
```

### Good

```text
LoyaltyPolicy.goldRate = 10%
LoyaltyPolicy.discountFor(customer, subtotal) returns { rate, amount }
Checkout records the returned discount decision
Invoice renders decision.rate and decision.amount
Refund reverses decision.amount
Change request: increase the Gold discount to 12%
Required edit: LoyaltyPolicy.goldRate and its contract test
```
