# Integration tests verify real boundaries

Integration tests exercise the real seams where components, adapters, schemas, infrastructure, or configuration meet.

Integration tests should prove that separately reasonable parts still work when connected.
They are most valuable at boundaries where unit tests cannot reveal reality: persistence, migrations, serialization, queues, HTTP contracts, auth middleware, file systems, configuration, and external adapters.
Strong signs include realistic dependencies where dependency behavior matters, owned test data, disposable or isolated infrastructure, clear setup and teardown, and assertions that catch wiring, contract, or infrastructure mistakes.
Weak signs include fake integrations that cannot catch the real class of defect, shared databases with mysterious state, tests that require manual environment preparation, and broad failure surfaces that make diagnosis guesswork.
This symptom matters because many production failures happen at seams rather than inside isolated logic.
Review should ask whether the integration test exercises the boundary honestly and fails with evidence that points toward the broken seam.

## Examples

### Bad
```text
test "repository saves a customer":
  database = FakeRows(acceptAnyColumn = true)
  repository = CustomerRepository(database, productionMapping)
  repository.save(Customer(email = "ada@example.test"))
  assert database.saved.email == "ada@example.test"
```

### Good
```text
test "repository saves a customer":
  database = DisposablePostgres(productionVersion)
  applyProductionMigrations(database)
  repository = CustomerRepository(database, productionMapping)
  repository.save(Customer(email = "ada@example.test"))
  assert repository.findByEmail("ada@example.test").email == "ada@example.test"
```
