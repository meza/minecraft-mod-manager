# Contributing

When contributing to this repository, please first discuss the change you wish to make via issue,
discussions, or any other method with the owners of this repository before making a change.

Please note we have a code of conduct, please follow it in all your interactions with the project.

## Pull Request Process

1. Ensure any install or build dependencies are removed (or ignored) before the end of the layer when doing a
   build (.idea, vscode, etc directories especially).
2. Make sure ALL your commits use the Conventional Commits specification.
3. Update the README.md with details of changes.
4. Make sure all the tests and linters pass.

## Quality bar (what good looks like)

This project optimizes for long-term maintainability and predictable behavior. Prefer small, boring, readable changes over clever
ones.

## Ways of working (contract)

This is how we contribute quality code.

### Test-first workflow (mandatory)

The expectation of the system MUST be expressed in proper automated tests first.

- Write the test first.
  - If the behavior is user-facing, capture the reviewed requirement as a third-person BDD scenario and assert observable outcomes. Use a snapshot only when complete layout or styling is part of the requirement.
- Verify the new test fails (prove the gap).
- Then, and only then, implement the code change that makes the test pass.

Completion means the tests prove it. If you cannot write a test for the expectation, stop and resolve that before proceeding.

## Shared infrastructure risk

This CLI relies on shared infrastructure and credentials. That means a single bad actor can degrade or remove service for
every user, and recovering may require a coordinated release. Treat any change that could amplify abuse, increase request
volume, or relax safeguards as a safety boundary.

Do not add user-facing knobs or configuration that let users change behavior in ways that could increase external load or
reduce protections without explicit approval. If you believe an exception is needed, open an issue and get sign-off before
changing behavior.

### Code philosophy

- Simplicity first: implement the smallest change that satisfies the requirement.
- Apply KISS. Seek smallest changes satisfying criteria
- Treat YAGNI as an active constraint. Require justification for abstractions.
- Maintainability: prefer clear control flow and explicit dependencies over deep abstraction.
- Testability: structure code so important behavior can be validated by fast, deterministic tests.
- Consistency: follow existing patterns in this repo. Do not introduce a new pattern unless it replaces an old one and is agreed
  ahead of time.
- Prioritize correctness and testability over cleverness
- The project is cross-platform (first-class support for Windows, macOS, Linux) for both users and developers.

### Naming and file conventions

We prefer descriptive names that stay clear outside Go-specific idioms.

- Receiver names should be descriptive and derived from the type (for example, `client *Client`). Avoid single-letter receivers.
- Avoid single-letter identifiers except `t`, `err`, `cfg`, `cmd`, and `ctx` in narrow scopes.
- Go source filenames must be lowercase. Use `snake_case` for multiword names.

### Testing philosophy (cross-cutting)

We treat automated tests as the primary contract for behavior and user experience.

- Prefer tests that exercise real production wiring and code paths.
- Use fakes/stubs only to control nondeterminism (time, random, network, filesystem, OS signals) or to force rare error paths. Do
  not stub core behavior to "make coverage green".
- Capture product requirements as third-person BDD scenarios and assert observable outcomes through the real MMM process.
- Execute shared BDD scenarios unchanged across the execution profiles. Follow the [testing architecture](docs/testing/README.md): drivers execute actions, shared product assertions verify capabilities, and separately owned presentation checks reuse suitable journeys.
- For equivalent outcomes, compare the durable records and relevant resolved decisions required by the [permanent transcript contract](docs/intent.md#active-display-and-permanent-transcript). Do not compare raw terminal-control bytes or input exchanges, and do not normalize away meaningful discrepancies.
- For terminal behavior, use tui-test for input, waits, screen state, process lifecycle, and snapshots. Do not add project-owned PTY or terminal emulation helpers.
- Prefer stable i18n keys, interpolation arguments, exit status, filesystem effects, and semantic terminal state over rendered wording.
- Use a full terminal snapshot only when the reviewed requirement depends on complete layout or styling. Update it only for an intentional product change.
- Every user-facing behavior change must be backed by at least one automated test that would fail if the behavior regressed.

Terminal interaction docs:
- Product requirements: [product intent](docs/intent.md) and [command guides](docs/commands/README.md)
- Developer guidance and examples: [terminal interaction corpus](docs/interactions/README.md)
- Verification workflow: [terminal E2E guide](docs/testing/terminal-harness.md)

### Required local checks

- The code adheres to engineering and code quality standards
- The changes don't re-invent the wheel by not using existing abstractions
- New features and bug fixes are covered by tests
- Product scenarios exist for user-visible behavior changes
- Required layout or styling behavior has explicit tui-test snapshot coverage at the relevant terminal dimensions
- All new code follows the established patterns in this repo
- All relevant documentation is updated
- `make fmt-check` passes - this verifies if `make fmt` has been run. If not, run `make fmt` to format the code.
- `make lint` passes (`make lint-fix` if lint reports fixes)
- `make vuln` passes
- `make coverage` passes (runs tests and enforces 100% coverage)
- `make build` passes
- `make e2e` passes when the change affects interactive terminal behavior

**IMPORTANT**

Run the repo `make` targets (do not call go test/go build directly):

HTTP-dependent terminal scenarios follow the agreed [HTTP fixture design](docs/testing/http-fixtures.md): scenario-owned Go HTTP servers and E2E-only endpoint overrides. This wiring is not implemented yet; the design's endpoint variables are not current runtime options.

The [existing VCR helper reference](docs/testing/http-vcr.md) describes the unused in-process helper and its recording workflow. It is not the E2E fixture boundary.

The current tagged terminal E2E suite requires the pinned tui-test CLI version. See the [terminal E2E guide](docs/testing/terminal-harness.md#current-cli-prerequisite) for installation and the `TUI_TEST_BIN` override. The agreed target is the [native Go binding](docs/testing/terminal-harness.md#agreed-native-go-integration); dependency pinning and harness migration are pending. Continue using the current CLI prerequisites and `make e2e` until that migration is implemented.

### Optional checks

- `make test-race` (slower, use before larger concurrency changes)

To update a retained non-E2E Go snapshot after an intentional change, run:

```bash
UPDATE_SNAPS=true make coverage
```

`make coverage` runs `go test ./...` as part of the unified coverage tool. It generates `coverage.html` from the filtered profile and writes the `go tool cover -func` output to `coverage.out` (filtered when exclusions are configured), then enforces 100% coverage.
`make lint` and `make lint-fix` always run the `golangci-lint` version pinned in `go.mod` via `go run`. The pinned tool dependency is declared in `tools.go`.

### Git hooks (optional)

We use lefthook to keep local hooks aligned with repo make targets.

Install lefthook and hooks:

```bash
go install github.com/evilmartians/lefthook/v2@v2.0.12
lefthook install
```

Current hooks run:

- `pre-commit`: `make lint`, `make coverage`
- `pre-push`: `make build`
- `post-merge`: `make mod-download`

### Packaging release artifacts

Use `make dist` to package existing build outputs into `dist/mmm-<os>-<arch>-<version>.zip` (defaults to `dev`).
The dist output includes `dist/metadata/THIRD_PARTY_NOTICES.txt` and `dist/metadata/mmm-sbom.json` generated by `make notices` and `make sbom` (they are not inside the zip files).
Use `make prepare` when you want both the build and packaging steps in one command.

### Release readiness for 3.0

[Product intent](docs/intent.md) defines the Go-port release behavior; the [command guides](docs/commands/README.md) explain its operator workflows. Delivery progress and work-item dispositions belong in the issue tracker. Historical feature-parity claims or closed epics alone do not establish conformance to the intended product.

Before release:

- Survey all command error paths before finalizing error-remediation scope, including paths outside existing scenarios. Verify that version-constraint failures identify the relevant constraint and are distinguished from missing projects and network failures. This preserves the comprehensive error-audit commitment associated with #452, including the reported connection-timeout problem in #349 if still present.
- Resolve or explicitly accept the findings of the 2025-12-19 independent audit, historically tracked as mmm-63. Record their dispositions in the tracker rather than claiming a current completion state in this document.
- Demonstrate the acceptance journeys in product intent, including manifest-cache outage recovery and proxy behavior. The cache and proxy commitments associated with #630 and #1030 remain in the 3.0 baseline.
- Verify CI on pull requests and pushes, and semantic-release automation on release branches. The pipeline must produce and publish the documented Windows, macOS and Linux archives, containing `mmm.exe` on Windows and `mmm` on macOS and Linux. Inject release credentials without exposing them in logs.
- Run release validation in the actual release context. A dry run must demonstrate the expected artifacts and release workflow before publishing, alongside evidence that the release requirements are met.

Historical parity work also mentioned update notifications without defining their behavior in product intent. Preserve that item for explicit tracker disposition before release; do not silently drop it or infer a new product contract from the old parity label.

### Translations and i18n

All user-facing strings must go through i18n. Translation authoring cost matters.

- Translation files live in `internal/i18n/lang/*.json`. Read `internal/i18n/README.md` before changing strings.
- Duplicated content is a translator tax. If 2 or more keys use identical user-facing content, reuse a single key instead of duplicating the same string under many keys.
- Only use multiple keys with identical English when the meaning is intentionally different and translations are expected to differ by locale. Treat this as an exception and justify it in the PR or work item.

## Product Work

This section explains how someone in a product capacity (Product Owner, product manager, or anyone doing requirements/product direction work) should interact with this project.

### State Management Philosophy

Product state lives in documentation and the issue tracker, not in ephemeral conversation or separate tracking systems. This ensures:

- **Portability**: Product knowledge travels with the codebase
- **Auditability**: Decisions are traceable and discoverable
- **Continuity**: Anyone can bootstrap product context by reading docs

### Where Product Artifacts Live

| Artifact Type                   | Location         | Purpose                          |
|---------------------------------|------------------|----------------------------------|
| Product expectations           | `docs/intent.md` | Authoritative product baseline   |
| Delivery plans and open questions | Issue tracker | Progress and stakeholder decisions |
| Requirements & success criteria | Issue tracker    | Trackable, closeable work items  |
| Command behavior and user guides | `docs/commands/` | Inputs, workflows, outcomes and examples |

### Issue Tracking

Use the issue tracker for:
- Feature requests with acceptance criteria
- Bug reports with reproduction steps
- Tasks with clear done conditions
- Open questions that need resolution

## Effort Estimation Vocabulary

This project uses relative effort sizing.

### T-Shirt Sizes

When estimating work effort, use these categories:

| Size                 | Meaning                                                                                                                                                                          |
|----------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **S (Small)**        | Minimal complexity. Well-understood problem space. Localized change with clear boundaries. Low risk of unexpected complications.                                                 |
| **M (Medium)**       | Moderate complexity. Some unknowns to resolve. May require coordination across a few components. Manageable scope with reasonable confidence.                                    |
| **L (Large)**        | Significant complexity. Multiple components affected or substantial unknowns. Requires careful design consideration. Higher likelihood of discovering additional scope.          |
| **XL (Extra Large)** | High complexity. Cross-cutting concerns or architectural impact. Major unknowns that may require exploration before implementation can begin. Should be broken down if possible. |

### What These Sizes Represent

T-shirt sizes express **relative effort and complexity**, not calendar time. They answer the question: "How complex is this work compared to other work we do?"

They do not answer: "How long will this take?" That question depends on capacity, availability, parallelization, and other factors outside the scope of effort estimation.

### Product Decisions (ADRs)

Significant product decisions that _significantly_ disrupt/alter the product, belong in `doc/adr/` alongside technical architecture decisions. Use ADR format when a decision:

- Affects multiple features or commands
- Establishes a pattern or precedent
- Resolves a trade-off that will recur
- Needs attribution (who decided, under what authority, why)

Before creating a product ADR, ask yourself if the decision truly needs formal documentation.
Many product decisions can be captured in issue comments or documentation updates instead.

### Authority Boundaries

Product Owners working on this project:

- **May decide independently** when the decision is underpinned by existing documentation or recorded outcomes in tickets/issues
- **Must surface for stakeholder input** any decision that lacks documented support or creates new precedent
- **Should not** read source code to determine product state. If documentation is insufficient, surface that gap rather than deriving answers from implementation
