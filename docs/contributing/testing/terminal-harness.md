# Terminal E2E testing with tui-test

This guide owns native terminal tooling, process lifecycle and evidence capture.
Start with the [testing guide](README.md) to choose between scenario authoring,
presentation checks and HTTP fixtures. Terminal tests use Microsoft's `tui-test`
Go binding, which calls the Rust engine in the test process:

```text
Gherkin scenario -> Godog step -> BDD actor action -> mode-specific driver -> tui-test Go binding -> Rust engine -> native MMM process
```

MMM runs as a separate native process in an isolated temporary workspace. Drivers
use the binding's public methods directly for input, waits and observations; they
contain no acceptance assertions. Follow [BDD architecture](bdd.md#runner-drivers-and-shared-assertions)
for runner, action, driver and assertion responsibilities.

## Prerequisites

Use Go 1.26 or newer, make, and a writable user cache directory that permits loading
native libraries. The binding package is `github.com/microsoft/tui-test/bindings/go`;
it loads its embedded native engine through `purego`.

Dependency maintenance must pin a reproducible binding revision containing matching
native artifacts for Windows, macOS and Linux. Do not use a developer-specific
module replacement or DLL path as contributor setup. Consumer builds need no
separate tui-test CLI, TypeScript runtime, Rust toolchain or C compiler; building
the engine itself is a separate dependency-maintenance task.

## Running the suite

Build the credential-free, host-native E2E binary and run the tagged scenarios:

```bash
make e2e
```

To build the tagged binary without running scenarios:

```bash
make e2e-build
```

The binary is written to `build/e2e`. `MMM_E2E_BINARY` can point the scenarios at another E2E-tagged native binary.

The scenario lifecycle supplies the [HTTP fixture endpoints](http-fixtures.md#endpoint-configuration)
to every MMM launch, including scenarios that do not use HTTP. For acceptance,
follow the [BDD verification requirements](bdd.md#run-and-extend-coverage).

## Writing terminal scenarios

Follow [BDD architecture](bdd.md) for feature placement, shared scenarios and
assertion ownership. Use [presentation testing](presentation.md) when a reviewed
requirement calls for visual evidence from a journey.

Prefer these observable outcomes:

- requested i18n keys and interpolation arguments;
- semantic terminal state exposed by tui-test;
- process exit status;
- files created, changed, or left untouched.

Use tui-test waits for text, idle state, and process exit, and its input, key, mouse
and resize methods for interaction. Use snapshots only when a reviewed requirement
depends on the complete rendered terminal state.

Keep terminal machinery in tui-test. Do not add project-owned PTYs, terminal
emulators, screen buffers, ANSI normalizers, frame extractors, input encoders,
polling loops, process supervisors, sleeps or compatibility wrappers.

## Rendering assertions and snapshots

Choose the observation that proves the requirement:

| Observation | Native API and scope |
| --- | --- |
| Visible layout, wrapping, clipping and stale content | `ExpectSnapshot` compares the visible terminal grid against a `.snap` baseline. |
| Colours and styling | `SnapshotOptions.IncludeColors` includes colour and style changes; `Cells` and `GetByStyle` support focused assertions. |
| Text placement and duplication | `GetByText` locators expose match positions and counts. |
| Resize behaviour | `Resize`, followed by state, text or snapshot assertions at the required dimensions. |
| Cursor position | `GetCursor`; assert separately from the grid snapshot. |
| Transcript preservation | `Text` with `TextOptions.Full` includes scrollback. |
| Interaction transitions | Keyboard and mouse operations, including scroll input, combined with waits and observations between actions. |
| Failure evidence | `Screenshot` can write SVG; recording and assertion artifacts retain additional diagnostics. |

Use focused semantic assertions for content and outcomes. Separately owned presentation checks use snapshots when the reviewed requirement depends on complete layout or styling; drivers only expose the observation path. Choose an explicit snapshot working directory with `SnapshotOptions.Cwd`; baselines live beneath it in `__snapshots__`. Fix the terminal dimensions, backend, locale and scenario data for each baseline. For translated wrapping and layout, exercise actual translations as well as localization-key mode.

Keep `SnapshotOptions.Update` disabled during normal verification. A missing baseline can be written with result `SnapshotWritten` even when update mode is disabled, so successful verification must require `SnapshotPassed` as well as no error. Create or update baselines deliberately, inspect the resulting diff, and retain them in version control. `IncludeTitle` is optional; enable it only when the title is part of the requirement.

Wait for an observable product state before capturing a frame. `WaitIdle` indicates a quiet screen, not operation completion. Use controlled HTTP fixture responses to make progress states repeatable; do not stabilize animations with arbitrary sleeps. A grid snapshot excludes scrollback and cursor state, and does not prove transcript preservation or terminal restoration. Those require the multi-stage observations below. Emulator snapshots also do not establish identical font rendering in every desktop terminal.

For cross-profile comparisons, follow [durable product assertions](bdd.md#product-and-presentation-assertions).
Exclude temporary active frames, terminal-control bytes and input exchanges;
do not strip text, reorder records or normalize away discrepancies.

## Stable localization expectations

For message-selection checks, launch the tagged E2E binary with `MMM_TEST=1`.
The presence of `MMM_TEST` enables key mode; `1` is the project convention.
Localization calls then emit the requested i18n key and interpolation arguments
instead of translated wording. This verifies message selection while keeping
scenarios stable when wording changes. For translated layout and input checks,
omit `MMM_TEST` from the child environment and select the locale being exercised.

Normal release binaries ignore `MMM_TEST` for localization. Go test binaries retain their existing key mode.

## Lifecycle and diagnostics

Each scenario gets:

- a unique `Ephemeral` tui-test client;
- an explicit working directory, environment, terminal size, and timeout;
- an isolated temporary workspace;
- an exact-session close and workspace cleanup in the scenario hook.

Godog owns scenario setup and cleanup. Register cleanup before launching MMM,
collect available failure diagnostics before closing, propagate `Close` failures,
and remove the workspace after the session is closed. Named clients share sessions
within the test process; do not use `CloseAll` or close unrelated sessions.

Native methods return Go errors, including typed `tuitest.Error` categories. They
block and session operations are serialized. Configure bounded native waits and an
overall suite timeout; do not assume context support or per-call cancellation.
Retain the original operation failure if optional diagnostic capture also fails.

When investigating a failure:

1. Identify the scenario, profile, failed operation and original error. Use the
   [result distinctions](presentation.md#lifecycle-and-results) to separate product,
   presentation, fixture/driver and unavailable-evidence outcomes.
2. Before closing the session, collect full terminal text, terminal state and the
   recording path when available. Retain text/SVG artifacts and recordings where useful.
3. Inspect text and state for the required observation; inspect a snapshot diff or
   SVG for layout failures. A final frame cannot establish an earlier animation,
   scroll transition or restoration. Reproduce those through the same driver with
   observations at the required points.
4. For HTTP-dependent failures, retain [fixture errors and request observations](http-fixtures.md#lifecycle-and-diagnostics).
   Complete cleanup even if capture fails, and report cleanup errors alongside the
   original failure.

## Verify terminal lifetimes

[Product intent](../../intent.md#acceptance-and-evidence) owns acceptance requirements. Use the [component examples](../interactions/component-examples.md) to discuss presentation, not as an automatic snapshot baseline. The following observations connect the rendering architecture to real terminal evidence:

| Journey | Evidence to retain |
| --- | --- |
| Transcript lifetime | Pre-command shell output and settled records survive exit exactly once in completion order, including failures, warnings, summaries and cancellation. Active repaints do not alter committed order; summaries do not replay items. Relevant resolved decisions appear neutrally regardless of whether prompts, arguments or defaults supplied them. |
| Scroll round trip | Scroll away while work continues, let results arrive and a prompt become pending, resize, then return. The reading position remains stable and the current active state returns. Reaching the active end resumes following. |
| Screen transitions | If the renderer uses an alternate screen, verify durable primary-screen history and terminal restoration as well as the visible active frame. |
| Shared capabilities | Exercise confirmation, selection, setup/correction and progress through multiple consuming commands, including preserved input and return after accepted recovery. |
| Safe cancellation | Observe ongoing cleanup, preserved completed work, the second-interruption warning and restored terminal control. A test timeout is a harness bound, not authorization for a production recovery timeout. |
| Profiles and localization | Exercise interactive TUI in Unicode and ASCII, plain line-oriented interaction without control sequences, explicit `--unattended`, and non-interactive execution such as redirected I/O. Verify that unattended and non-interactive runs never prompt and contain no ANSI or cursor-control output. Supplying complete arguments alone must not select unattended. Exercise real translated input and layout in addition to localization-key checks. |

Drive input, scrolling, resize, waits and lifecycle through tui-test. Use deterministic cross-process [HTTP fixtures](http-fixtures.md) for network-dependent journeys. Do not reintroduce Expect wrappers or custom probe replies for discovery; use the same terminal driver for investigation and acceptance.

Record the platform, terminal dimensions and relevant capabilities with the evidence. A captured frame alone does not prove scrollback or restored shell state. Observe a host shell session when a requirement concerns output before invocation or control after process exit. If a required observation is unavailable, identify the verification gap and use supported tui-test capabilities rather than replacing the observation with model output. Do not claim acceptance until the required evidence exists.

When a failure concerns rendering ownership or terminal restoration, consult the
[terminal implementation guide](../guide-to-working-with-the-terminal.md). When
establishing a control or presentation expectation, consult the
[interaction conventions](../interactions/interaction-guidelines.md).
