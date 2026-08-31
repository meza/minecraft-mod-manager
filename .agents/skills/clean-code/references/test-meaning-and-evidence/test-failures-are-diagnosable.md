# Test failures are diagnosable

When tests fail, the output gives enough context to understand the broken behavior without guesswork or habitual reruns.

A good test failure should point toward the violated behavior and the scenario that produced it.
The more moving parts a test has, the more important its failure evidence becomes.
Strong signs include clear test names, specific assertions, useful assertion messages, captured logs or traces where appropriate, visible request and response details at boundaries, and artifacts for end-to-end failures.
Weak signs include vague failures such as expected true to be false, swallowed setup errors, enormous undifferentiated logs, missing screenshots or traces for UI failures, and failure messages that force the reviewer to reproduce locally before understanding anything.
This symptom matters because undiagnosable failures turn automated tests into delay rather than evidence.
Review should ask whether a failing test would help the next engineer find the likely cause.

## Examples

### Bad

```text
assertTrue(discountedTotal(memberOrder) == 80)
Failure: expected true, got false
```

### Good

```text
expectedTotal = 80
actualTotal = discountedTotal(memberOrder)
assertEqual(actualTotal, expectedTotal, "member order total after discount")
Failure: member order total after discount: expected 80, got 100
```
