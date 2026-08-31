# Test meaning and evidence

The tests state and protect meaningful promises about behavior, cover invariants and unhappy paths, and give a reader enough evidence to trust a pass and diagnose a failure.

Use the descriptions below to decide which individual quality references are relevant. Open only the references that can materially affect the current decision, code, or failure.

- [Tests encode behavior, not implementation trivia](test-meaning-and-evidence/tests-encode-behavior-not-implementation-trivia.md) - Tests describe externally meaningful promises rather than internal structure that should be free to change.
- [Important invariants are tested](test-meaning-and-evidence/important-invariants-are-tested.md) - Each important rule, guarantee, or failure mode has at least one test that would fail if the guarantee broke.
- [Happy and unhappy paths are both tested](test-meaning-and-evidence/happy-and-unhappy-paths-are-both-tested.md) - Negative cases, edge cases, and misuse paths are intentionally exercised rather than left implicit.
- [Tests are readable](test-meaning-and-evidence/tests-are-readable.md) - A reviewer can tell what behavior a test protects and why it matters.
- [Test setup is cheap to understand](test-meaning-and-evidence/test-setup-is-cheap-to-understand.md) - Creating the conditions for a test does not require navigating a maze of fixtures, mocks, or magical helpers.
- [Over-mocking is avoided](test-meaning-and-evidence/over-mocking-is-avoided.md) - Tests do not become brittle by asserting every internal interaction instead of meaningful outcomes.
- [Regression evidence exists](test-meaning-and-evidence/regression-evidence-exists.md) - Changed behavior is backed by tests or checks that prove the intended effect and protect against relapse.
- [Manual-only verification is minimized](test-meaning-and-evidence/manual-only-verification-is-minimized.md) - Important correctness claims are not left resting on ad hoc human checking when they could be mechanized.
- [Evidence backs correctness claims](test-meaning-and-evidence/evidence-backs-correctness-claims.md) - Assertions that something works are supported by checks, tests, logs, traces, commands, or measurable evidence.
- [Test failures are diagnosable](test-meaning-and-evidence/test-failures-are-diagnosable.md) - When tests fail, the output gives enough context to understand the broken behavior without guesswork or habitual reruns.
- [Test scope matches test name and intent](test-meaning-and-evidence/test-scope-matches-test-name-and-intent.md) - The name, setup, action, and assertions of a test all describe the same behavior at the same level of abstraction.

