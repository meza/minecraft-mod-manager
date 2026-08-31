# Test design reference

## Test observable behavior

Use a public interface and assert an independently known result.

```go
func TestCartCalculatesTotal(t *testing.T) {
	cart := NewCart()
	cart.Add(Item{Price: 10})
	cart.Add(Item{Price: 5})

	assert.Equal(t, 15, cart.Total())
}
```

The literal `15` is an independent worked result. Recomputing the expected value with the production algorithm would make the test tautological.

## Use durable names

Name the behavior, not the delivery activity.

```go
// Avoid: the ticket identifies the work rather than the behavior.
func TestMMM228Item18(t *testing.T) {}

// Prefer: the name remains useful after the ticket closes.
func TestRestorePreservesStoredStatus(t *testing.T) {}
```

Keep ticket IDs, pull request numbers, and review item numbers in delivery records rather than executable specifications.

## Avoid implementation coupling

A test is implementation-coupled when it mocks internal collaborators, exercises private functions instead of public behavior, or asserts call order that callers cannot observe. Such tests fail during harmless refactoring and may remain green when the public behavior is broken.

Prefer exercising the real public path. Replace only nondeterministic system boundaries.

## Control time explicitly

When behavior depends on the current time, inject a clock or current-time function and freeze it before arranging the subject under test.

```go
fixedNow := time.Date(2026, time.August, 31, 12, 0, 0, 0, time.UTC)
service := NewService(func() time.Time { return fixedNow })
```

Derive relative inputs such as expired, future, age, and elapsed time from the frozen clock. A date literal remains appropriate when that exact date is a domain input and the behavior does not interpret it relative to now.

## Treat mechanical verification separately

Do not commit a test that scans the repository merely to prove that a requested string, file, or identifier was removed. Record a targeted search as completion evidence instead:

```bash
rg -n '\bObsoleteName\b' .
```

Use permanent lint or static analysis only when the absence is an explicit enduring repository policy.
