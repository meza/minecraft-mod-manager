# Controlling test boundaries

Replace dependencies only at system boundaries that introduce nondeterminism or effects, such as:

- external APIs;
- time or randomness;
- filesystems and operating-system signals; and
- databases when a real test database is not proportionate.

Do not mock internal collaborators merely to expose implementation details. Prefer testing the real production path through its public interface.

## Inject narrow interfaces

Accept the capability the subject needs instead of constructing a concrete external client internally.

```go
type Doer interface {
	Do(request *http.Request) (*http.Response, error)
}

type Client struct {
	doer Doer
}
```

Tests can then provide a deterministic `Doer`, while production uses the real HTTP implementation. Keep fakes behavior-specific and free of conditional logic unrelated to the case under test.

Prefer an existing project boundary over introducing a new interface. Add a seam only when the architecture needs it and the behavior cannot be tested reliably through an existing public path.
