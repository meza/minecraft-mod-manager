# Architectural integrity is preserved across domains and interfaces

The architecture keeps a clear organizing principle, respects domain boundaries, preserves intent across APIs and integrations, resists drift, enforces system-wide rules structurally, and remains organized around domain meaning rather than utility sprawl.

Architectural integrity is the property that lets a large system still feel like one designed thing instead of a pile of locally reasonable edits. A clear center of gravity, respected bounded contexts, intent-preserving APIs, stable integration contracts, resistance to drift, structural enforcement of system-wide rules, and organization around domain rules all belong to one lens because they determine whether the architecture keeps teaching the same story as the codebase grows.

Strong signs include an obvious organizing principle for where behavior belongs; domain concepts that stay within their own context unless explicitly translated; APIs and service boundaries that expose business meaning instead of database or transport leakage; integrations that remain stable unless a real contract change has occurred; conventions enforced through types, tests, generators, templates, or architecture rather than mere style guidance; and code organization dominated by capabilities and domain rules rather than a sprawling helper layer. Weak signs include context leakage, adapters that expose storage detail as public contract, recurring structural exceptions, silent erosion of intended layering, and utility modules that become the real center of gravity of the system.

This symptom matters because architecture usually fails gradually. A single change rarely destroys it, but many small changes can make the structure stop reflecting the domain. Review this lens by asking whether the change strengthens or weakens the long-running integrity of the system's organizing principles and interface boundaries. A poor score here suggests that local convenience is winning over architectural coherence.

## Examples

### Bad

```text
Billing.renew(subscriptionId):
  subscription = Subscriptions.database.find(subscriptionId)
  discount = 0
  if subscription.planCode == "ANNUAL" and subscription.failedPayments == 0:
    discount = subscription.monthlyPrice * 0.10
  return charge(subscription.accountId, subscription.monthlyPrice - discount)
```

### Good

```text
interface SubscriptionRenewals:
  quote(subscriptionId) -> RenewalQuote
Subscriptions.quoteRenewal(subscriptionId):
  subscription = repository.get(subscriptionId)
  return renewalPolicy.quote(subscription)
RenewalCheckout.renew(subscriptionId):
  quote = subscriptionRenewals.quote(subscriptionId)
  return billing.charge(quote.accountId, quote.amount)
```
