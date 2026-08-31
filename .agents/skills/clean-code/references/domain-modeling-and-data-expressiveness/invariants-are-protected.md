# Invariants are protected

The code structure makes invalid states difficult or impossible to represent or persist.

The design should help preserve truths the domain depends on. Good code uses types, constructors, validation, and encapsulation so that invalid combinations are blocked structurally instead of being remembered socially. In review, this is about how much interpretation work the reader must do before they can trust what the code is trying to accomplish. Strong signals are names, structure, and local flow that let another engineer build a correct mental model quickly. Weak signals are vague labels, mixed levels of abstraction, hidden assumptions, or a need to chase many references before the unit makes sense. The educational point is that readability is not cosmetic; it is what makes future change, debugging, and review safe. For this specific symptom, the reviewer should ask whether the change makes 'Invariants are protected' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
class Account
  public balance

account = Account(balance: 100)
account.balance = -50
```

### Good

```text
class Account
  private balance
  constructor(openingBalance)
    require openingBalance >= 0
    balance = openingBalance
  withdraw(amount)
    require 0 < amount <= balance
    balance = balance - amount
```
