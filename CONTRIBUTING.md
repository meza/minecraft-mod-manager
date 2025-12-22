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

### Code philosophy

- Simplicity first: implement the smallest change that satisfies the requirement.
- Apply KISS. Seek smallest changes satisfying criteria
- Treat YAGNI as active constraint; require justification for abstractions
- Maintainability: prefer clear control flow and explicit dependencies over deep abstraction.
- Testability: structure code so important behavior can be validated by fast, deterministic tests.
- Consistency: follow existing patterns in this repo; do not introduce a new pattern unless it replaces an old one and is agreed
  ahead of time.
- Prioritize correctness and testability over cleverness
- The project is cross-platform (first-class support for Windows, macOS, Linux) for both users and developers.

### Testing philosophy (cross-cutting)

We treat automated tests as the primary contract for behavior and user experience.

- Prefer tests that exercise real production wiring and code paths.
- Use fakes/stubs only to control nondeterminism (time, random, network, filesystem, OS signals) or to force rare error paths; do
  not stub core behavior to "make coverage green".
- Snapshot tests are the primary guardrail against UX regressions:
  - For any user-visible output (TUI or non-TUI), add snapshot coverage of the rendered output.
  - When possible, drive TUIs with `teatest` and snapshot the output/view so we catch regressions in interaction and presentation.
- Every user-facing behavior change must be backed by at least one automated test that would fail if the behavior regressed.

### Required local checks

Run the repo `make` targets (do not call go test/go build directly):

- `make fmt`
- `make lint`
- `make lint-fix`
- `make coverage` (runs tests and enforces 100% coverage)
- `make build`

### Optional checks

- `make test-race` (slower; use before larger concurrency changes)

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
Use `make prepare` when you want both the build and packaging steps in one command.

## Product Work

This section explains how someone in a product capacity (Product Owner, product manager, or anyone doing requirements/product direction work) should interact with this project.

### State Management Philosophy

Product state lives in documentation and the issue tracker—not in ephemeral conversation or separate tracking systems. This ensures:

- **Portability**: Product knowledge travels with the codebase
- **Auditability**: Decisions are traceable and discoverable
- **Continuity**: Anyone can bootstrap product context by reading docs

### Where Product Artifacts Live

| Artifact Type                   | Location              | Purpose                                 |
|---------------------------------|-----------------------|-----------------------------------------|
| Roadmaps, PRDs, open questions  | `docs/product/`       | Product Owner working documents         |
| Requirements & success criteria | Issue tracker (beads) | Trackable, closeable work items         |
| Command specifications          | `docs/specs/`         | Detailed behavioral specs               |
| User-facing documentation       | `docs/commands/`      | How users interact with features        |

### Issue Tracking

For the Go port, issue tracking uses **beads** (`.beads/` directory, accessed via `bd` CLI).

Use the issue tracker for:
- Feature requests with acceptance criteria
- Bug reports with reproduction steps
- Tasks with clear done conditions
- Open questions that need resolution

### Product Decisions (ADRs)

Significant product decisions that _significantly_ disrupt/alter the product, belong in `doc/adr/` alongside technical architecture decisions. Use ADR format when a decision:

- Affects multiple features or commands
- Establishes a pattern or precedent
- Resolves a trade-off that will recur
- Needs attribution (who decided, under what authority, why)

Before creating a product ADR, ask yourself if the decision truly needs formal documentation.
Many product decisions can be captured in issue comments or documentation updates instead.

### The `docs/product/` Directory

This directory is for product artifacts that don't fit elsewhere. Use it sparingly—most product work belongs in:

- **Issue tracker**: Requirements, success criteria, bugs, tasks
- **`docs/specs/`**: Detailed behavioral specifications
- **`docs/commands/`**: User-facing documentation

Only add to `docs/product/` when an artifact genuinely doesn't fit those locations. Examples: roadmaps, open questions awaiting resolution, or PRDs for larger features that span multiple issues.

### Authority Boundaries

Product Owners working on this project:

- **May decide independently** when the decision is underpinned by existing documentation or recorded outcomes in tickets/issues
- **Must surface for stakeholder input** any decision that lacks documented support or creates new precedent
- **Should not** read source code to determine product state—if documentation is insufficient, surface that gap rather than deriving answers from implementation
