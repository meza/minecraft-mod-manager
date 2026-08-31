# Smoke tests are broad, shallow, and release-gating

Smoke tests quickly determine whether a build, deployment, or environment is healthy enough for deeper testing or use.

Smoke tests are not regression suites.
They should answer whether the application starts, is reachable, is configured well enough to perform its core role, and has not obviously broken its most critical path.
Strong signs include fast checks after deployment, basic reachability, real readiness rather than superficial liveness, core dependency connectivity, safe idempotent behavior, and failure that stops more expensive pipeline stages.
Weak signs include smoke suites that take so long they become miniature regression suites, checks that only prove a process is alive while the product is unusable, tests that mutate shared state unsafely, and smoke results that are ignored by delivery.
This symptom matters because smoke tests are the pipeline's early warning system.
Review should ask whether the smoke suite is small enough to trust and meaningful enough to block waste.

## Examples

### Bad

```text
deploy checkout-api to staging
run the four-hour checkout regression suite
test every coupon boundary and payment retry sequence
record smoke failures but continue the pipeline
promote the staging release to production
```

### Good

```text
deploy checkout-api to staging
assert readiness reports database and payment gateway ready
assert a shopper can sign in
assert a shopper can place one idempotent test order
assert the order can be read back
if any assertion fails, stop the release
promote the staging release to production
```
