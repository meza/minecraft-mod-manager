# Test placement and isolation

Each test sits at the cheapest layer that can provide trustworthy evidence and is insulated from the world, so results reflect product behavior rather than environment.

Use the descriptions below to decide which individual quality references are relevant. Open only the references that can materially affect the current decision, code, or failure.

- [Integration points are exercised realistically](test-placement-and-isolation/integration-points-are-exercised-realistically.md) - At least some tests cover real wiring between modules, adapters, and system boundaries.
- [Tests are deterministic](test-placement-and-isolation/tests-are-deterministic.md) - Test outcomes do not depend on timing luck, ordering accidents, external state, or flaky infrastructure.
- [Tests sit at the right level](test-placement-and-isolation/tests-sit-at-the-right-level.md) - Behavior is verified at the cheapest test layer that can provide trustworthy evidence, rather than pushed unnecessarily into broad slow tests.
- [Unit tests avoid physical I/O](test-placement-and-isolation/unit-tests-avoid-physical-i-o.md) - Unit tests verify focused behavior without touching the real filesystem, network, database, clock, or other physical/environmental resources unless that resource is the behavior under test. Open this when assessing what a test actually touches and whether it belongs at the unit-test layer.
- [Integration tests verify real boundaries](test-placement-and-isolation/integration-tests-verify-real-boundaries.md) - Integration tests exercise the real seams where components, adapters, schemas, infrastructure, or configuration meet.
- [End-to-end tests protect critical journeys](test-placement-and-isolation/end-to-end-tests-protect-critical-journeys.md) - End-to-end tests cover a small number of high-value user or system journeys through the deployed shape of the application.
- [Smoke tests are broad, shallow, and release-gating](test-placement-and-isolation/smoke-tests-are-broad-shallow-and-release-gating.md) - Smoke tests quickly determine whether a build, deployment, or environment is healthy enough for deeper testing or use.
- [Test data is owned and isolated](test-placement-and-isolation/test-data-is-owned-and-isolated.md) - Tests create, control, and clean up the data and state they depend on rather than relying on accidental shared conditions.
- [External dependencies are controlled in tests](test-placement-and-isolation/external-dependencies-are-controlled-in-tests.md) - Tests bound or replace nondeterministic third-party systems so failures reflect product behavior rather than outside instability.
- [Unit tests mock physical boundaries](test-placement-and-isolation/unit-tests-mock-physical-boundaries.md) - Unit tests avoid real filesystem, network, database, clock, process, and environment access unless the physical boundary itself is the behavior under test. Open this when assessing whether production design exposes a substitutable seam for a physical dependency.

