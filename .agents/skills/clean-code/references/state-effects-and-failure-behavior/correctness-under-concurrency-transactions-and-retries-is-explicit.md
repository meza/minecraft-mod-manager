# Correctness under concurrency, transactions, and retries is explicit

The code makes ordering, atomicity, and safe re-entry understandable so concurrency, transaction handling, and repeated execution do not create hidden correctness risks.

Some of the hardest defects come from behavior that is only wrong when timing changes, retries happen, or multiple operations interleave. Concurrency model clarity, transaction boundary clarity, and idempotence all belong to one review lens because they describe whether the system remains correct when work does not happen exactly once in a simple straight line.

Strong signs include explicit ownership of shared mutable state, clear synchronization or scheduling assumptions, well-defined transaction boundaries around integrity-sensitive work, and operations that are intentionally safe or intentionally guarded when duplicate messages, retries, or repeated calls can occur. Weak signs include hidden races, ambiguous ordering requirements, transactions that are much wider or narrower than the invariants they are meant to protect, side effects that can be repeated accidentally, and code that works only if calls happen once and in a socially remembered sequence.

This symptom matters because many systems are retrying, asynchronous, distributed, or simply used by multiple actors at once. Review should therefore ask not only whether the happy path is correct, but whether the implementation stays correct when execution overlaps, partially fails, retries, or resumes. Good code makes those assumptions inspectable. Bad code leaves them implicit and turns correctness into luck.

## Examples

### Bad

```text
function withdraw(requestId, accountId, amount):
  account = accounts.get(accountId)
  if account.balance < amount: reject
  account.balance -= amount
  accounts.save(account)
  result = withdrawals.insert(requestId, accountId, amount)
  mailer.sendReceipt(result)
  return result
```

### Good

```text
function withdraw(requestId, accountId, amount):
  transaction:
    if withdrawals.contains(requestId): return withdrawals.get(requestId)
    account = accounts.getForUpdate(accountId)
    if account.balance < amount: reject
    account.balance -= amount
    result = withdrawals.insertUnique(requestId, accountId, amount)
    outbox.insertUnique(requestId, ReceiptRequested(requestId, result))
  return result
on ReceiptRequested(e): mailer.send(e.result, idempotencyKey=e.requestId)
```
