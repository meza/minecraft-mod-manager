# Shared rules have a single owner

Core rules are defined in one authoritative place rather than being re-stated in multiple modules and tests.

A rule should have one authoritative home. If the same decision logic is reimplemented in multiple places, bugs arise when one copy changes and another does not. In review, this is about changeability at module, boundary, and architectural level rather than only about local cleanliness. Strong signals are explicit dependencies, narrow interfaces, coherent module ownership, and boundaries that reflect real responsibilities in the system. Weak signals are reach-through access, policy mixed with mechanism, accidental framework-driven structure, and changes that would force broad unrelated edits. The educational point is that design quality shows up most clearly when requirements evolve or when one part of the system fails. For this specific symptom, the reviewer should ask whether the change makes 'Shared rules have a single owner' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
Checkout.discountFor(customer):
  if customer.years >= 3 and customer.spend >= 1000: return 0.10
  return 0
Renewals.discountFor(customer):
  if customer.years >= 3 and customer.spend >= 1000: return 0.10
  return 0
test "loyal customers get 10%":
  customer = Customer(years: 3, spend: 1000)
  eligible = customer.years >= 3 and customer.spend >= 1000
  assert eligible and Checkout.discountFor(customer) == 0.10
```

### Good

```text
LoyaltyDiscount.rateFor(customer):
  if customer.years >= 3 and customer.spend >= 1000: return 0.10
  return 0
Checkout.discountFor(customer): return LoyaltyDiscount.rateFor(customer)
Renewals.discountFor(customer): return LoyaltyDiscount.rateFor(customer)
test "loyal customers get 10%":
  customer = Customer(years: 3, spend: 1000)
  assert LoyaltyDiscount.rateFor(customer) == 0.10
```
