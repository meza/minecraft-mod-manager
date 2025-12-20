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
- `make test`
- `make coverage-enforce`
- `make build`

### Packaging release artifacts

Use `make dist` to package existing build outputs into `dist/mmm-<os>-<arch>-<version>.zip` (defaults to `dev`).
Use `make prepare` when you want both the build and packaging steps in one command.
