# Test scope matches test name and intent

The name, setup, action, and assertions of a test all describe the same behavior at the same level of abstraction.

A test should tell one coherent story.
Its name should describe the condition and expected behavior, its setup should create only the relevant scenario, its action should exercise the behavior under review, and its assertions should check the promised outcome.
Strong signs include names that read like behavior specifications, setup that does not introduce unrelated facts, one primary reason for failure, and assertions aligned with the stated purpose.
Weak signs include vague names, tests that assert several unrelated outcomes, setup that hides the important condition, names that claim business behavior while assertions check implementation detail, and broad tests whose real purpose is impossible to summarize.
This symptom matters because unclear test intent makes review weaker and maintenance more dangerous.
Review should ask whether the test would still make sense if read as documentation of the protected behavior.

## Examples

### Bad

```text
test "expired coupon is rejected":
  coupon = Coupon(active = true)
  result = checkout.apply(coupon)
  assert audit.events contains "coupon_applied"
```

### Good

```text
test "expired coupon is rejected":
  coupon = Coupon(expired = true)
  result = checkout.apply(coupon)
  assert result == Rejected("coupon expired")
```
