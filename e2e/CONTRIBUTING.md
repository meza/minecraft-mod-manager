# Contributing E2E tests

Follow the [repository contribution guide](../CONTRIBUTING.md). Start with the
[testing guide](../docs/testing/README.md) for architecture, examples and tooling.

- Execute each shared capability scenario unchanged in every execution profile,
  with isolated scenario state, process lifecycle, fixtures and workspace.
- Keep steps and actions about the product. Drivers use tui-test's Go binding to
  perform interactions and expose evidence; they contain no product-outcome or
  presentation assertions.
- Keep product assertions shared. Attach presentation checks to suitable existing
  journeys without adding visual requirements to shared Gherkin or copying the flow.
  Both kinds of required checks must pass.
- Follow [BDD architecture](../docs/testing/bdd.md) for responsibilities and the
  [feature contribution guide](features/CONTRIBUTING.md) for authoring rules.
- Follow [presentation testing](../docs/testing/presentation.md) for spinner,
  layout and other presentation evidence, timing and separate failure reporting.
- Use the [terminal harness guide](../docs/testing/terminal-harness.md) for current
  prerequisites, `make e2e`, lifecycle and diagnostics. It distinguishes the available
  CLI adapter from the required native Go-binding design.
