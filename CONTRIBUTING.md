# Contributing

When contributing to this repository, please first discuss the change you wish to make via issue,
discussions, or any other method with the owners of this repository before making a change.

Please note we have a code of conduct, please follow it in all your interactions with the project.

## Pull Request Process

1. Ensure any install or build dependencies are removed before the end of the layer when doing a
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
  - If the behavior is user-facing, the test MUST be a snapshot test that captures the rendered user-visible output.
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
- Snapshot tests are the primary guardrail against UX regressions (hard requirement):
  - All user-facing behavior paths MUST be covered by snapshot tests. If a user can observe a difference, it needs a snapshot.
  - For any user-visible output (interactive terminal (tui-lite) or non-interactive terminal), add snapshot coverage of the rendered output.
  - If it has an interface (a tui-lite screen/view/prompt/menu/table), it MUST have snapshot tests that cover all branches and states of
    the UI.
  - "All branches and states" includes (at minimum): success, empty/no results, loading, validation errors, recoverable errors,
    fatal errors, and any conditional rendering (for example: selected vs unselected, focused vs unfocused, enabled vs disabled,
    expanded vs collapsed, pagination).
  - When possible, drive tui-lite flows with `teatest` and snapshot the output/view so we catch regressions in interaction and presentation.
  - Update snapshots only when the user-visible behavior is intentionally changed.
- Every user-facing behavior change must be backed by at least one automated test that would fail if the behavior regressed.

Terminal interaction docs:
- Behavior contract: `docs/interactions/interaction-guidelines.md`
- Implementation guide: `docs/guide-to-working-with-the-terminal.md`

### Required local checks

- The code adheres to engineering and code quality standards
- The changes don't re-invent the wheel by not using existing abstractions
- New features and bug fixes are covered by tests
- Snapshots exist for all user-visible behavior changes
- Complete snapshot tests exist for short (25 rows) and tall (80 rows) terminal heights
- All new code follows the established patterns in this repo
- All relevant documentation is updated
- `make fmt-check` passes (`make fmt` if not formatted)
- `make lint` passes (`make lint-fix` if lint reports fixes)
- `make vuln` passes
- `make coverage` passes (runs tests and enforces 100% coverage)
- `make build` passes

**IMPORTANT**

Run the repo `make` targets (do not call go test/go build directly):

If you need to record HTTP cassettes for scenario tests, use `make vcr-record`.
See `docs/testing/http-vcr.md` for the workflow and cassette conventions.

### Optional checks

- `make test-race` (slower, use before larger concurrency changes)

To update snapshots (when you change user-visible output), run:

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
| Roadmaps, PRDs, open questions  | `docs/product/`  | Product Owner working documents  |
| Requirements & success criteria | Issue tracker    | Trackable, closeable work items  |
| Command specifications          | `docs/specs/`    | Detailed behavioral specs        |
| User-facing documentation       | `docs/commands/` | How users interact with features |

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

### The `docs/product/` Directory

This directory is for product artifacts that don't fit elsewhere. Use it sparingly as most product work belongs in:

- **Issue tracker**: Requirements, success criteria, bugs, tasks
- **`docs/specs/`**: Detailed behavioral specifications
- **`docs/commands/`**: User-facing documentation

Only add to `docs/product/` when an artifact genuinely doesn't fit those locations. Examples: roadmaps, open questions awaiting resolution, or PRDs for larger features that span multiple issues.

### Authority Boundaries

Product Owners working on this project:

- **May decide independently** when the decision is underpinned by existing documentation or recorded outcomes in tickets/issues
- **Must surface for stakeholder input** any decision that lacks documented support or creates new precedent
- **Should not** read source code to determine product state. If documentation is insufficient, surface that gap rather than deriving answers from implementation
