# Terminal E2E testing with tui-test

Terminal end-to-end tests currently use Microsoft's `tui-test` CLI. The agreed target is the native Go binding, which calls the Rust engine in the test process. Migration of the Godog harness is not implemented yet. The project does not own a PTY, terminal emulator, screen buffer, ANSI normalizer, input encoder, polling loop, or process supervisor.

The current test boundary is:

```text
Gherkin scenario -> Godog step -> BDD actor action -> tui-test CLI -> native MMM process
```

The native process runs in an isolated temporary workspace. Assertions observe terminal state, exit status, and filesystem effects.

## Agreed native Go integration

The target boundary is:

```text
Gherkin scenario -> Godog step -> BDD actor action -> mode-specific driver -> tui-test Go binding -> Rust engine -> native MMM process
```

Godog owns scenario setup and cleanup. Each mode-specific driver uses the binding's public methods directly for supported process operations, input, waits and observations. The binding hosts the terminal engine in the Go test process; MMM remains a separately launched process in an isolated workspace. The [BDD architecture](bdd.md#runner-drivers-and-shared-assertions) owns runner, action, driver and product-assertion responsibilities. Drivers contain no acceptance assertions; [presentation checks](presentation.md) can assess evidence from the same invocation separately from product checks. HTTP fixtures retain their separate [scenario-owned server boundary](http-fixtures.md).

### Dependency setup and migration status

The evaluated package is `github.com/microsoft/tui-test/bindings/go`. Its Go API loads an embedded native engine through `purego`. The evaluated binding requires Go 1.26 or newer and a writable user cache directory that permits loading native libraries. A package containing the matching engine needs no tui-test CLI, TypeScript runtime, Rust toolchain or C compiler at consumer build time. Building the engine itself is a separate dependency-maintenance task.

Before migration, select and pin a reproducible binding revision with matching native artifacts for each supported platform. MMM has not pinned this dependency yet; the local evaluation does not establish a published release or a portable installation procedure. Do not commit a developer-specific module replacement or DLL path as the contributor setup. The intended suite entry point remains `make e2e`; the CLI prerequisite below remains necessary until migration lands.

The Windows amd64 evaluation on 5 September 2026 exercised real MMM launch, resize, loader-prompt detection, Ctrl+C, exit status 0, absence of configuration files, and session close. It also passed with `CGO_ENABLED=0`. This establishes binding feasibility for that smoke journey, not acceptance of every product cancellation requirement. Native snapshot execution, scroll/restoration journeys, and macOS/Linux operation against MMM remain unverified.

## Current CLI prerequisite

Install tui-test `0.1.0-beta.2` from the [Microsoft tui-test repository](https://github.com/microsoft/tui-test). The E2E adapter rejects a different version or JSON schema with an actionable error.

The adapter looks for `tui-test` on `PATH` and in the standard installer locations. To use another executable, set `TUI_TEST_BIN`:

```bash
TUI_TEST_BIN=/path/to/tui-test make e2e
```

PowerShell:

```powershell
$env:TUI_TEST_BIN = 'C:\path\to\tui-test.exe'
make e2e
```

Ordinary Go tests do not require tui-test.

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

HTTP-dependent scenarios will use the agreed [HTTP fixture design](http-fixtures.md), with scenario-owned Go servers and E2E-only endpoint overrides. That wiring is not implemented yet; the endpoint variables in the design are not current runtime options. `make e2e` continues to run the existing suite.

## Writing terminal scenarios

Follow [BDD architecture](bdd.md) for shared scenarios, profile selection and assertion ownership. Put product scenarios in `e2e/features` and reusable actions and outcomes in the `e2e` package. Keep Gherkin about what an actor does and observes; attach visual checks according to [presentation testing](presentation.md) instead of embedding them in shared scenarios or drivers.

Prefer these observable outcomes:

- requested i18n keys and interpolation arguments;
- semantic terminal state exposed by tui-test;
- process exit status;
- files created, changed, or left untouched.

Use tui-test waits for text, idle state, and process exit. Use its input, key, mouse, and resize commands for interaction. Use tui-test snapshots only when a reviewed requirement depends on the complete rendered terminal state.

Do not add sleeps, ANSI cleanup, frame extraction, terminal buffers, key encoders, PTY code, or compatibility wrappers. For current-suite work, use capabilities exposed by the existing CLI adapter. The agreed migration uses the Go binding's public API for terminal operations.

## Rendering assertions and snapshots

The evaluated native binding exposes the following capabilities. Their presence in the API does not establish coverage in MMM's current suite.

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

Keep `SnapshotOptions.Update` disabled during normal verification. The evaluated engine writes a missing baseline and returns `SnapshotWritten` even when update mode is disabled, so successful verification must require `SnapshotPassed` as well as no error. Create or update baselines deliberately, inspect the resulting diff, and retain them in version control. `IncludeTitle` is optional; enable it only when the title is part of the requirement.

Wait for an observable product state before capturing a frame. `WaitIdle` indicates a quiet screen, not operation completion. Use controlled HTTP fixture responses to make progress states repeatable; do not stabilize animations with arbitrary sleeps. A grid snapshot excludes scrollback and cursor state, and does not prove transcript preservation or terminal restoration. Those require the multi-stage observations below. Emulator snapshots also do not establish identical font rendering in every desktop terminal.

When profiles should produce equivalent outcomes, compare their durable records under the same locale and character capabilities. Exclude temporary active frames, terminal-control bytes and the input exchange itself from that comparison. Do not strip text, reorder records or otherwise normalize away discrepancies. Relevant decisions must remain visible in neutral language whether they were supplied by a prompt answer, an argument or a documented default.

## Stable localization expectations

The tagged E2E binary is launched with `MMM_TEST=1`. The presence of `MMM_TEST` enables key mode; `1` is the project convention. Localization calls then emit the requested i18n key and interpolation arguments instead of translated wording. This verifies that the product requests the correct message while keeping scenarios stable when wording changes.

Normal release binaries ignore `MMM_TEST` for localization. Go test binaries retain their existing key mode.

## Lifecycle and diagnostics

Each scenario gets:

- a uniquely named tui-test session;
- an explicit working directory, environment, terminal size, and timeout;
- an isolated temporary workspace;
- an exact-session close and workspace cleanup in the scenario hook.

On failure, the adapter reports the failed CLI operation and collects the recording path, full terminal text, and terminal state before cleanup. It never closes unrelated tui-test sessions.

The native migration must preserve these responsibilities using a unique `Ephemeral` client per scenario. Register cleanup before launching MMM, collect available failure diagnostics before closing, propagate `Close` failures, and remove the workspace after the session is closed. Named clients share sessions within the test process; do not use `CloseAll` for scenario cleanup.

Native methods return Go errors, including typed `tuitest.Error` categories. They block, operations on a session are serialized, and the evaluated API does not accept contexts or promise per-call cancellation. Configure bounded native waits and preserve an overall suite timeout; do not assume the existing CLI subprocess cancellation translates to native calls. Use text/SVG failure artifacts and recordings where useful, retaining the original operation failure if optional capture also fails.

## Historical tests

The removed custom harness and its expectations are catalogued in the [legacy terminal test ledger](legacy-terminal-test-ledger.md). The ledger is historical evidence, not an approved product specification. Do not recreate its snapshots as tui-test baselines without reviewing the requirement first.

## Verify terminal lifetimes

[Product intent](../intent.md#acceptance-and-evidence) owns acceptance requirements. Use the [component examples](../interactions/component-examples.md) to discuss presentation, not as an automatic snapshot baseline. The following observations connect the rendering architecture to real terminal evidence:

| Journey | Evidence to retain |
| --- | --- |
| Transcript lifetime | Pre-command shell output and settled records survive exit exactly once in completion order, including failures, warnings, summaries and cancellation. Active repaints do not alter committed order; summaries do not replay items. Relevant resolved decisions appear neutrally regardless of whether prompts, arguments or defaults supplied them. |
| Scroll round trip | Scroll away while work continues, let results arrive and a prompt become pending, resize, then return. The reading position remains stable and the current active state returns. Reaching the active end resumes following. |
| Screen transitions | If the renderer uses an alternate screen, verify durable primary-screen history and terminal restoration as well as the visible active frame. |
| Shared capabilities | Exercise confirmation, selection, setup/correction and progress through multiple consuming commands, including preserved input and return after accepted recovery. |
| Safe cancellation | Observe ongoing cleanup, preserved completed work, the second-interruption warning and restored terminal control. A test timeout is a harness bound, not authorization for a production recovery timeout. |
| Profiles and localization | Exercise interactive TUI in Unicode and ASCII, plain line-oriented interaction without control sequences, explicit `--unattended`, and non-interactive execution such as redirected I/O. Verify that unattended and non-interactive runs never prompt and contain no ANSI or cursor-control output. Supplying complete arguments alone must not select unattended. Exercise real translated input and layout in addition to localization-key checks. |

Drive input, scrolling, resize, waits and lifecycle through tui-test. Use deterministic cross-process fixtures for network-dependent journeys, following the agreed [HTTP fixture design](http-fixtures.md). Do not reintroduce Expect wrappers, custom probe replies, sleeps or terminal emulation for discovery; use the same terminal driver for investigation and acceptance.

Record the platform, terminal dimensions and relevant capabilities with the evidence. A captured frame alone does not prove scrollback or restored shell state. Observe a host shell session when a requirement concerns output before invocation or control after process exit. If the current adapter cannot expose a required observation, identify that verification gap and use supported tui-test capabilities rather than replacing the observation with model output. Do not claim acceptance until the required evidence exists.

Also read the [BDD E2E guide](bdd.md), the [terminal implementation guide](../guide-to-working-with-the-terminal.md), and the [interaction conventions](../interactions/interaction-guidelines.md).
