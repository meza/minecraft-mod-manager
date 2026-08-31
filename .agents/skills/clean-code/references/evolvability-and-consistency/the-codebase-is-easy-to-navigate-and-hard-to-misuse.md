# The codebase is easy to navigate and hard to misuse

Repository shape, API design, and local structure help contributors find the right place to work, guide them toward correct usage, and make incorrect usage conspicuous or difficult.

A maintainable system should help engineers do the right thing by default. Cheap navigation, misuse-resistant design, and ease of correct use belong together because they describe whether the codebase teaches safe participation. A reader should be able to locate responsibility quickly, understand the intended path through an API or module, and notice when a usage pattern is likely wrong.

Strong signs include repository and module layouts that make likely locations easy to predict, interfaces whose shape guides callers toward valid combinations and sequences, defaults that are safe, and types or contracts that make misuse visible. Weak signs include sprawling structures where responsibility is hard to find, APIs that expose many invalid states or surprising combinations, and "correct" usage that requires oral tradition while the easy path does something risky or misleading.

This symptom matters because a large share of maintenance cost comes from accidental misuse and search friction, not just defects in core logic. Review this lens by asking whether the change helps the next contributor orient quickly and follow the intended path safely, or whether it adds more ways to get lost, call the wrong thing, or violate an assumption without noticing.

## Examples

### Bad

```text
// shared/utils/money
function adjust(accountId, cents, reverse)
adjust(order.customerId, order.totalCents, true)
```

### Good

```text
// billing/refunds
function refund(order: PaidOrder): RefundReceipt
refunds.refund(order)
```
