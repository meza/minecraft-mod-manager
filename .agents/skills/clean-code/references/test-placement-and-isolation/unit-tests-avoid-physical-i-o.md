# Unit tests avoid physical I/O

Unit tests verify focused behavior without touching the real filesystem, network, database, clock, or other physical/environmental resources unless that resource is the behavior under test.

A good unit test gives precise evidence about a small piece of behavior while staying fast, deterministic, and isolated.
Filesystem access should normally be abstracted behind an interface, adapter, fake, in-memory filesystem, or mock so the unit can be tested without physical I/O.
Physical filesystem access, including OS temporary files or directories, is acceptable only when the unit's behavior genuinely depends on filesystem semantics that a mock would hide, such as permissions, path handling, locking, atomic writes, symlink behavior, or real encoding and newline behavior.
Strong signs include one clear behavior per test, direct assertions about outputs or visible effects, explicit edge cases, simple setup, mocked or in-memory file access, injected clocks and randomness, and no dependency on prior test execution.
Weak signs include casual use of tmpfiles for convenience, tests that read or write ambient paths, global state leaks, real network or database access, excessive mock choreography, assertions that duplicate implementation logic, and tests of private implementation detail.
This symptom matters because unit tests should be cheap, deterministic, and safe to run constantly.
Review should ask whether any physical I/O in the unit test is essential to the behavior being proven, or whether it should be mocked, faked, or moved to an integration test.

## Examples

### Bad

```text
test "expired cache refreshes report" [misplaced broad test: filesystem, wall clock, live HTTP]:
  files = OSFileSystem(tempDirectory())
  clock = SystemClock()
  api = HttpReportApi("https://reports.example.test")
  files.write("report.cache", cached("old", expires = clock.now() - 1 second))
  report = ReportService(files, api, clock).current()
  assert report == "new"
```

### Good

```text
test "expired cache refreshes report" [unit]:
  files = InMemoryFileSystem()
  clock = FixedClock("2030-01-01T00:00:00Z")
  api = FakeReportApi(returning = "new")
  files.write("report.cache", cached("old", expires = clock.now() - 1 second))
  report = ReportService(files, api, clock).current()
  assert report == "new"
  assert api.calls == 1
```
