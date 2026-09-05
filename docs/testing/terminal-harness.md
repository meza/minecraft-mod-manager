# Terminal E2E testing with tui-test

Terminal end-to-end tests use Microsoft's `tui-test` CLI. The project does not own a PTY, terminal emulator, screen buffer, ANSI normalizer, input encoder, polling loop, or process supervisor.

The test boundary is:

```text
Gherkin scenario -> Godog step -> BDD actor action -> tui-test CLI -> native MMM process
```

The native process runs in an isolated temporary workspace. Assertions observe terminal state, exit status, and filesystem effects.

## Prerequisite

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

## Writing terminal scenarios

Put product scenarios in `e2e/features` and reusable actions and outcomes in the `e2e` package. Keep the Gherkin in third person and express what an actor does and observes.

Prefer these observable outcomes:

- requested i18n keys and interpolation arguments;
- semantic terminal state exposed by tui-test;
- process exit status;
- files created, changed, or left untouched.

Use tui-test waits for text, idle state, and process exit. Use its input, key, mouse, and resize commands for interaction. Use tui-test snapshots only when a reviewed requirement depends on the complete rendered terminal state.

Do not add sleeps, ANSI cleanup, frame extraction, terminal buffers, key encoders, PTY code, or compatibility wrappers. Extend the thin CLI adapter only when tui-test already provides the capability through its public JSON interface.

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

## Historical tests

The removed custom harness and its expectations are catalogued in the [legacy terminal test ledger](legacy-terminal-test-ledger.md). The ledger is historical evidence, not an approved product specification. Do not recreate its snapshots as tui-test baselines without reviewing the requirement first.

Also read the [BDD E2E guide](bdd.md), the [terminal implementation guide](../guide-to-working-with-the-terminal.md), and the [interaction guidelines](../interactions/interaction-guidelines.md).
