# Test data is owned and isolated

Tests create, control, and clean up the data and state they depend on rather than relying on accidental shared conditions.

Trustworthy tests own their preconditions.
A test should not pass because a developer's machine, a shared staging database, a previous test, or a long-lived account happens to contain the expected state.
Strong signs include explicit data creation, isolated schemas or namespaces, disposable containers or fixtures, unique identifiers, rollback or cleanup, and tests that can run repeatedly or in parallel without collision.
Weak signs include dependence on magic records, shared mutable accounts, hidden ordering requirements, manual preloading, tests that only work on one machine, and data left behind that changes later outcomes.
This symptom matters because uncontrolled test data is one of the fastest ways for useful automation to become flaky noise.
Review should ask whether the test can prove the same thing from a clean starting point.

## Examples

### Bad

```text
test "suspend customer":
  customer = findCustomer("customer-42")
  suspend(customer)
test "suspended customer cannot checkout":
  customer = findCustomer("customer-42")
  assert checkout(customer).error == "suspended"
```

### Good

```text
test "suspended customer cannot checkout":
  customer = createCustomer(id = uniqueId(), status = "active")
  try:
    suspend(customer)
    assert checkout(customer).error == "suspended"
  finally:
    deleteCustomer(customer)
```
