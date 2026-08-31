# Assumptions, intent, and structural decisions are explicit

The code and its surrounding artifacts make assumptions, change intent, and important architectural decisions visible enough that reviewers and future maintainers do not have to reconstruct them from guesswork.

Software becomes expensive to maintain when key reasoning is trapped in someone's head or in an old conversation. Labeling assumptions, making change intent reviewable, and surfacing architectural decisions belong together because they all reduce invisible policy. They help a reviewer answer three basic questions: what does this change assume, what is it trying to accomplish, and what larger structural choice does it depend on or introduce.

Strong signs include explicit notes where assumptions materially affect correctness, commit or change context that explains why a shape was chosen, and discoverable records or code-level signals for decisions that constrain later design. Weak signs include code that only makes sense if the reader already knows unstated operational facts, structural changes with no visible rationale, and diffs that provide no clue which behavior is being protected or traded away.

This symptom matters because clarity of reasoning is part of maintainability. Future reviewers should not be forced to infer every hidden premise from implementation details alone. Review this lens by asking whether the change teaches the next engineer how to reason about it, or whether it hides assumptions and design intent behind local code that only appears obvious right now.

## Examples

### Bad

```text
function importOrders(orders):
    transaction:
        for order in orders:
            save(order)
        markBatchComplete()
```

### Good

```text
# Assumption: one worker owns a batch; save preserves input order.
# Intent: downstream readers must never observe a partial batch.
# Decision: commit the orders and completion marker atomically.
function importOrders(orders):
    transaction:
        for order in orders:
            save(order)
        markBatchComplete()
```
