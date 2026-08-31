# Names are domain-specific and precise

Names use the language of the domain and distinguish meaning clearly enough that readers do not have to infer intent from surrounding context.

Good naming carries both familiarity and precision. The best names use the vocabulary that the surrounding domain, tests, and documentation already use, while also being specific enough to distinguish one idea, responsibility, or state from another. Review is not only about whether a name sounds reasonable in isolation, but whether it reduces interpretation work for the next reader.

Strong signs include names that map cleanly to business concepts, technical concepts, or established system terms; names that make different responsibilities visibly different; and identifiers that remain clear even when read away from the call site. Weak signs include generic placeholders such as manager, data, process, handle, util, or thing-like labels that hide what the code actually represents; names that depend heavily on comments to become understandable; and parallel terms that appear to mean the same thing or the same term reused for different meanings.

This symptom matters because names are part of the interface of every unit, even when the code is private. Imprecise or non-domain naming increases review friction, makes change risk harder to see, and encourages incorrect assumptions to spread. Maintainable code tends to let a reader predict behavior from names before reading the implementation. When this symptom is weak, the code becomes harder to search, harder to discuss, and easier to misuse.

## Examples

### Bad

```text
function process(data):
  result = gateway.handle(data)
  if result.ok: ledger.save(result)
```

### Good

```text
function renewSubscription(expiredSubscription):
  renewalReceipt = paymentGateway.chargeRenewal(expiredSubscription)
  if renewalReceipt.approved: subscriptionLedger.recordRenewal(renewalReceipt)
```
