# External dependencies are controlled in tests

Tests bound or replace nondeterministic third-party systems so failures reflect product behavior rather than outside instability.

External systems introduce instability, cost, latency, and failure modes that may not belong in ordinary test feedback.
Payment providers, email systems, identity providers, analytics, remote APIs, clocks, random generators, and network services should be real only when the test is explicitly proving that integration.
Strong signs include sandboxed integrations, contract tests, fakes that preserve meaningful semantics, injected clocks or randomness, network boundaries that are deliberate, and clear separation between ordinary CI tests and tests that depend on live services.
Weak signs include uncontrolled calls to third-party systems, tests that fail because someone else's service was slow, real emails or payments sent by automation, and mocks that are so fake they remove the risk being tested.
This symptom matters because a test suite should identify defects, not outsource its signal to environmental luck.
Review should ask whether every external dependency in the test is either controlled or intentionally under test.

## Examples

### Bad

```text
test "checkout charges the card":
  receipt = checkout(LivePaymentProvider(), order)
  assert receipt.status == "paid"
```

### Good

```text
test "checkout charges the card":
  payments = ContractVerifiedFake(PaymentProviderContract, charge = approved)
  receipt = checkout(payments, order)
  assert receipt.status == "paid"
```
