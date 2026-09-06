# Contributing

Agree the requested outcome and scope with the project owners before making a change. An already
agreed request satisfies this step. Follow the [code of conduct](CODE_OF_CONDUCT.md) in project
interactions.

## Choose the route for your work

Read the relevant guides below and any README.md or CONTRIBUTING.md in the affected directory and
its ancestors. Combine routes for mixed changes; unrelated guides do not need to be loaded.

| Work | Required guidance |
| --- | --- |
| Product requirements or design | [Product work](#product-work), [product intent](docs/intent.md), [glossary](docs/GLOSSARY.md), and the relevant [command guide](docs/commands/README.md) |
| Go behavior or refactoring | [Code conventions](#code-conventions), [testing policy](docs/contributing/testing/policy.md), and affected command or package guides |
| Test-only changes | [Testing policy](docs/contributing/testing/policy.md), then the guide for the test boundary |
| Terminal components | [Terminal architecture](docs/contributing/guide-to-working-with-the-terminal.md), [Charm TUI guide (Bubble Tea, Bubbles and Lip Gloss)](docs/contributing/charm-tui/README.md), [interaction conventions](docs/contributing/interactions/README.md), and [component verification](docs/contributing/testing/components.md) |
| Stories or gallery integration | [Charm TUI guide](docs/contributing/charm-tui/README.md), [gallery contribution guide](tools/bubblebook/README.md), and [component verification](docs/contributing/testing/components.md) |
| E2E scenarios or harnesses | [E2E testing index](docs/contributing/testing/README.md) and its task-selected guides |
| Translations or user-facing text | [Translations](#translations), [i18n guide](internal/i18n/README.md), and [interaction conventions](docs/contributing/interactions/interaction-guidelines.md) for controls and confirmation tokens |
| Provider integrations | [Shared platform boundary](internal/platform/README.md) and the relevant [CurseForge](internal/curseforge/README.md) or [Modrinth](internal/modrinth/README.md) guide |
| Performance or telemetry | [Performance instrumentation](internal/perf/README.md) or [telemetry](internal/telemetry/README.md), according to the changed contract |
| Documentation or comments | [Documentation procedure](docs/contributing/documentation.md) and [comment policy](docs/contributing/code-comments.md) when comments change |
| Tooling, dependencies or configuration | [Cross-platform tooling](docs/contributing/cross-platform-tooling.md), affected documentation, and [verification](#verification) |
| Review or re-review | [Code review](docs/contributing/reviewing-code/README.md) and its task-selected procedure |
| Addressing review findings | [Review remediation](docs/contributing/reviewing-code/addressing-findings.md) |
| Commits or pull requests | [Submission](#submission) |
| Packaging or release | [Release guide](docs/contributing/releases.md) |
| Project instruction maintenance | [Document ownership and routing](docs/contributing/documentation.md#instruction-and-contribution-guidance) |

## Setup

Documentation and product work do not require the application toolchain. For Go development, use
the Go version and toolchain declared by the project, GNU Make, and your platform's terminal. Run
commands from the repository root. Download module dependencies with:

~~~sh
make mod-download
~~~

Use the applicable checks under [verification](#verification) to validate the change. The
[terminal harness prerequisites](docs/contributing/testing/terminal-harness.md#prerequisites) apply before
running native terminal E2E tests; the [gallery guide](tools/bubblebook/README.md) owns gallery
setup.

Distribution builds require the three API token defaults described in
[using your own API keys](README.md#using-your-own-api-keys). Keep credentials local and out of
commits, output, logs and fixtures. Do not obtain or expose production credentials merely to
perform documentation checks.

Those prerequisites also apply to the required `make build` check for Go changes. If suitable local
token defaults are unavailable, report that check as unverified. A development-only build does not
replace the required distribution-build evidence or authorize obtaining production credentials.

### Optional Git hooks

Lefthook provides local hooks aligned with repository make targets. To opt in:

~~~sh
go install github.com/evilmartians/lefthook/v2@v2.0.12
lefthook install
~~~

The documented hooks run `make lint` and `make coverage` before commit, `make build` before push,
and `make mod-download` after merge. Hooks are optional and may run more checks than a particular
surface needs; they do not replace the verification policy below.

## Code conventions

Optimize for long-term maintainability and predictable behavior. Implement the smallest coherent
change that satisfies the requirement. Justify new abstractions; prefer clear control flow,
explicit dependencies and fast, deterministic tests over cleverness.

Follow established patterns that fit the documented target architecture. A new pattern must replace
an old one and be agreed before introduction. Correctness and testability take priority. Windows,
macOS and Linux are first-class environments for users and developers.

- Use descriptive receiver names derived from the type, such as `client *Client`.
- Avoid single-letter identifiers except `t`, `err`, `cfg`, `cmd`, and `ctx` in narrow scopes.
- Go source filenames must be lowercase, using `snake_case` for multiword names.
- Follow the [comment policy](docs/contributing/code-comments.md) when adding or changing comments.

Investigate lint diagnostics and correct their cause. Do not weaken rules to make a change pass.
Suppress a diagnostic only when there is no sound fix, such as an unavoidable external naming
constraint; keep the exception local to the affected line and explain it. Improve inconsistent rules
through an agreed change; ask for help after investigating a diagnostic you cannot resolve.

## Shared infrastructure risk

This CLI relies on shared infrastructure and credentials. A single bad actor can degrade or remove
service for every user, and recovery may require a coordinated release. Treat changes that amplify
abuse, increase request volume or relax safeguards as a safety boundary.

Do not add user-facing knobs or configuration that increase external load or reduce protections
without explicit approval from the project owners. Obtain that approval before changing behavior.

## Verification

Classify each changed surface before choosing checks. Mixed changes require the applicable
combination. This policy applies to implementation and review; a reviewer records failures without
running formatting or autofix commands.

| Changed surface | Required evidence |
| --- | --- |
| Production behavior | Follow [test-first development](docs/contributing/testing/policy.md#behavior-changes), prove the expected failure and subsequent passing behavior, and run the Go gates below |
| Behavior-preserving refactoring | Existing tests protect the affected contract; run the Go gates without inventing a new behavior solely to add a test |
| Go test-only changes | Assert meaningful outcomes and run the relevant suite plus the Go gates; test review does not require reconstructing authoring chronology |
| Documentation | Check content, terminology, examples, links and consistency using the [documentation procedure](docs/contributing/documentation.md); no production tests or application build are required |
| Configuration or dependencies | Validate relevant syntax, schema, defaults, precedence and intended effects; run code or integration gates when the change affects those surfaces |
| Mechanical changes | Use targeted searches, diffs and relevant static checks; retain code gates if executable behavior or compilation may be affected |
| Development tooling | Apply the [tooling policy](docs/contributing/cross-platform-tooling.md), relevant static/build checks and tool-specific effect validation |

### Go gates

For production Go and Go test changes, all of these must pass:

- `make fmt-check`
- `make lint`
- `make vuln`
- `make coverage`, including the 100% coverage requirement
- `make build`

Implementers may use `make fmt` or `make lint-fix` to correct a failed check, then rerun the check.
Do not call Go test or build commands directly; use repository make targets.

Every user-facing behavior change requires a product scenario and at least one automated test
that would fail for a meaningful regression. Run `make e2e` when the change affects interactive
terminal behavior or E2E scenarios and harnesses. Use the [E2E index](docs/contributing/testing/README.md) for
required prerequisites and evidence, including other execution profiles affected by the change.
Required complete layout or styling behavior needs tui-test snapshot coverage at relevant terminal
dimensions. Direct component rendering checks complement that application evidence.

Run `make test-race` for effectful terminal-component changes. It is also available as an optional
additional check before larger concurrency changes. Report an unavailable race toolchain rather
than treating a non-race run as equivalent. Follow [component verification](docs/contributing/testing/components.md)
for component, root, runtime and story checks.

Required evidence that cannot be obtained remains unverified. Explain the constraint and its effect
on the completion claim; do not silently substitute an unrelated passing check or remote CI status.

## Translations

All user-facing strings go through i18n. Read the [i18n guide](internal/i18n/README.md) before changing
`internal/i18n/lang/*.json` or the strings that use it.

Reuse one key for identical user-facing content. Multiple keys with identical English are allowed
only when their meanings intentionally differ and translations are expected to differ by locale;
explain that exception in the change description. For interactive labels and confirmation tokens,
also follow [interaction conventions](docs/contributing/interactions/interaction-guidelines.md).

## Product work

[Product intent](docs/intent.md) defines the desired product and acceptance baseline. The
[glossary](docs/GLOSSARY.md) owns vocabulary; [command guides](docs/commands/README.md) own inputs,
workflows and outcomes. Keep durable product decisions in their owning documentation.

Product contributors may decide within existing documented requirements and approved outcomes.
Surface decisions that lack documented support or establish new precedent for stakeholder input.
Do not derive product requirements from source code when documentation is insufficient; identify
the missing product decision.

Significant, enduring architecture decisions belong in [the decision records](doc/adr/decisions.md).
A decision affecting several features or establishing a recurring precedent merits considering a
record, not automatically creating one. Keep routine or localized reasoning in the nearest owning
document; do not require an architecture record for ordinary delivery work.

## Submission

Every commit must follow Conventional Commits. These messages supply release metadata and version
changes; submissions that do not follow the specification are not accepted.

Before submitting:

- Keep local editor, installation and build artifacts out of the change, removing or ignoring them
  as appropriate.
- Explain the requested outcome, material changes, risks and relevant verification evidence.
- Update the documentation that owns the changed contract. Update README when its user-facing
  information changes; not every contribution needs a README edit.
- Complete the checks required for the changed surfaces.
- Keep secrets out of commits, logs, output and fixtures.

Use the [review guide](docs/contributing/reviewing-code/README.md) for an advisory review and the
[release guide](docs/contributing/releases.md) for packaging and release readiness.
