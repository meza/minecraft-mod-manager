# Terminal test harness

This guide shows how to write deterministic terminal interaction tests using the shared harnesses in `testutil/terminal`.
Use it when you need to snapshot Bubble Tea output, drive keyboard or mouse input, or validate PTY behavior.

## Quick start

Use the PTY harness when you need real terminal semantics like control sequences or alt screen behavior.

```go
func TestScanPromptPTY(t *testing.T) {
	terminal.ApplyFixtures(t)

	session := terminalpty.NewSession(t, terminalpty.WithSize(terminal.Size{Columns: 120, Rows: 12}))
	require.NotNil(t, session)

	cmd := commandUnderTest()
	cmd.SetIn(session.Input())
	cmd.SetOut(session.Output())
	cmd.SetErr(session.Output())

	execErr := make(chan error, 1)
	go func() {
		execErr <- cmd.Execute()
	}()

	session.WaitForOutput(t, func(data []byte) bool {
		normalized := terminal.NormalizeOutput(string(data), terminal.NormalizeOptions{
			StripControlSequences: true,
		})
		return strings.Contains(normalized, "cmd.scan.prompt.add")
	})
	// ApplyFixtures sets MMM_TEST=true, so prompt tokens are i18n keys.
	yesShort := i18n.T("cmd.init.prompt.option.yes.short", nil)
	_, writeErr := session.SendInput([]byte(yesShort + "\r"))
	require.NoError(t, writeErr)

	require.NoError(t, <-execErr)
	require.NoError(t, session.Close())

	output := terminal.NormalizeOutput(session.OutputString(), terminal.NormalizeOptions{
		StripControlSequences:  true,
		TrimTrailingWhitespace: true,
		TrimTrailingEmptyLines: true,
		TrimSpace:              true,
	})
	snaps.MatchSnapshot(t, output)
}
```

## Harness choices

### In-process harness (teatest)

Use `testutil/terminal/teatest` when you want fast, deterministic, in-process tests for Bubble Tea models.
It runs the program in-process and captures the exact writer output.
Prefer it for model-level snapshots where you can drive the state machine directly and do not need raw control sequences.

### PTY harness

Use `testutil/terminal/pty` when you need real terminal behavior, including control sequences, cursor moves,
alt screen behavior, mouse tracking, and resize semantics.
It captures raw bytes from a real PTY master.
Prefer it for integration-level tests that must validate renderer behavior, cursor positioning, mouse motion, or resize handling.

## Snapshot conventions

### Capture all states explicitly

Snapshot every stable state a user can see in the flow.
Drive the model through each step with explicit inputs and take a snapshot after each state transition.
Avoid a single snapshot at the end of the flow if intermediate prompts or progress states are user-visible.

In practice:
- For Bubble Tea models in-process, send a message, wait for the output to change, then snapshot.
- For PTY flows, wait for a known output token (like a prompt key) before sending input.

### Cover short and tall terminal sizes

Terminal rendering changes with height and width, so snapshot both:
- Short: 25 rows.
- Tall: 80 rows (or the platform-specific tall height used by the existing PTY tests).

Use the same sizes already present in the command PTY integration tests to keep snapshots consistent across commands.

### Keep snapshots deterministic

Normalize output only when needed for stability, and keep the normalization in the harness layer.
If you strip control sequences, do it consistently across tests and document why.
When MMM_TEST is enabled, prompt tokens and localized output become stable keys instead of natural language.

### Prefer explicit frame boundaries

If a TTY flow renders frames, snapshot a full frame rather than partial output.
Use helpers that trim to the last frame when the output includes earlier frames in the same buffer.

## Shared fixtures

Call `terminal.ApplyFixtures(t)` at the start of terminal tests.
It sets `MMM_TEST=true`, disables color by default, and locks unicode support to deterministic values.
Use options when you need a specific color profile or unicode behavior.

If you need a temporary `modlist.json`/`modlist-lock.json` setup for tests, see `testutil/modlistfixture/README.md` for the helper API and examples.

If your scenario makes external HTTP calls, call `vcr.LoadCassette` in the test to attach a cassette.
See `docs/testing/http-vcr.md` for the workflow and cassette conventions.
Because `terminal.ApplyFixtures` imports `vcr`, live external HTTP is blocked by default unless a cassette is active.

```go
terminal.ApplyFixtures(t,
	terminal.WithColorProfile(termenv.TrueColor),
	terminal.WithUnicodeEnabled(true),
)
```

## TTY and non-TTY setup

Use `WithCapabilities` to control terminal detection explicitly.
This is how you force non-TTY output in an in-process test.

```go
session := teatest.NewSession(t, model,
	teatest.WithCapabilities(terminal.NonTTYCapabilities()),
)
```

## Size and resize

Set the initial terminal size explicitly in tests.
Use `terminal.Size` for both harnesses.

```go
session := terminalpty.NewSession(t, terminalpty.WithSize(terminal.Size{Columns: 120, Rows: 12}))
```

When you need resize behavior, use the harness APIs:

```go
session.Resize(terminal.Size{Columns: 120, Rows: 40})
```

## Input simulation

### Keyboard

For in-process tests, send Bubble Tea messages directly:

```go
session.SendKey(tea.KeyMsg{Type: tea.KeyEnter})
```

For PTY tests, write raw bytes to the master:

```go
_, writeErr := session.SendInput([]byte("q"))
```

### Mouse

Use `terminal.EncodeSGRMouseSequence` to build PTY mouse input.
The helper encodes SGR mouse sequences that Bubble Tea understands.

```go
sequence := terminal.EncodeSGRMouseSequence(terminal.MouseEvent{
	Button: tea.MouseButtonWheelDown,
	Action: tea.MouseActionPress,
	X:      10,
	Y:      5,
})
_, writeErr := session.SendInput([]byte(sequence))
```

## Output capture and normalization

Use `OutputString` or `OutputBytes` to snapshot output.
When you need stable snapshots, normalize with `terminal.NormalizeOutput`.

```go
normalized := terminal.NormalizeOutput(session.OutputString(), terminal.NormalizeOptions{
	StripControlSequences:  true,
	TrimTrailingWhitespace: true,
	TrimTrailingEmptyLines: true,
	TrimSpace:              true,
})
```

## Waiting for output

Use `WaitForOutput` to avoid races before sending input.
It polls the captured output until a predicate is true.

```go
session.WaitForOutput(t, func(data []byte) bool {
	return strings.Contains(string(data), "cmd.scan.header.results")
})
```

If you just need to wait for a known token before closing the PTY, use `WaitForOutputAndClose` to reduce boilerplate.

```go
session.WaitForOutputAndClose(t, func(data []byte) bool {
	return strings.Contains(string(data), "cmd.scan.header.results")
})
normalized := terminal.NormalizeOutput(session.OutputString(), terminal.NormalizeOptions{
	StripControlSequences:  true,
	TrimTrailingWhitespace: true,
	TrimTrailingEmptyLines: true,
	TrimSpace:              true,
})
snaps.MatchSnapshot(t, normalized)
```

## FAQ

### When should I use teatest vs PTY?

Use teatest for model-level snapshots and state-machine coverage.
Use PTY for integration snapshots that must validate real terminal behavior (control sequences, alt screen, mouse, and resize semantics).
If you are unsure, start with teatest and move to PTY when you need terminal-level fidelity.

### How do I make sure teatest snapshots cover all states?

Treat each interactive state as a separate snapshot.
Send the exact message that triggers the state transition, wait for output to change, then snapshot the view.
For multi-step flows, keep a snapshot per step rather than only the final output.

### How do I ensure output renders correctly across terminal sizes?

Set explicit sizes in tests and cover both short and tall terminals.
Use the same sizes that the existing command PTY integration tests use, so snapshots remain consistent and comparable.
If a flow is sensitive to width, add at least one narrow width snapshot as well.

### Why do prompt tokens look like keys?

`terminal.ApplyFixtures(t)` sets `MMM_TEST=true`, which causes i18n to return keys instead of localized text.
Use the key strings when sending input (for example, `cmd.init.prompt.option.yes.short`) so the prompt parser accepts the input.

### Do I always need to normalize output?

No. Prefer raw output when it is stable.
Normalize only when control sequences or trailing whitespace make snapshots noisy.
Keep normalization consistent for a given test class so diffs are meaningful.

## Where the harness lives

- Shared fixtures and helpers: `testutil/terminal`
- In-process harness: `testutil/terminal/teatest`
- PTY harness: `testutil/terminal/pty`

You can stop here if you only need the harness API.
If you are adding new interaction scenarios, also review `docs/guide-to-working-with-the-terminal.md`
and `docs/interactions/interaction-guidelines.md`.
