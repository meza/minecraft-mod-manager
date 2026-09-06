# Component gallery

Use the gallery to preview MMM's shared visual primitives in isolation. It runs
[Bubblebook](https://github.com/sarkarshuvojit/bubblebook) as a separate developer
program, using the same primitives as the application.

## Run the gallery

You need Go (the version and toolchain declared in the root `go.mod`), GNU Make,
and an interactive terminal. Run this from the repository root:

```sh
make bubblebook
```

Go downloads missing module dependencies on the first run. The gallery itself
uses local sample data. It does not need API keys or a Minecraft installation.
Start with a terminal at least 120 columns wide and 30 rows high to leave room
for the component list and preview.

The gallery contains an animated spinner and progress variants starting at 0%,
50% and 100%. These appear in sample mod rows rendered through MMM's shared
presentation code, including its label and status styles. Progress uses a fixed
sample total; it does not download files.
Selecting another story creates a fresh instance, so returning to a progress
story restores its starting value.

Preview colors follow MMM's terminal capability checks and respect `NO_COLOR`.
If previews appear unstyled, check whether your shell sets `NO_COLOR`. To inspect
colors, unset it in that shell and restart the gallery in a color-capable terminal.

## Navigate and interact

| Key | Action |
| --- | --- |
| Up or Down | Select a story while the component list has focus |
| Tab | Switch focus between the list and preview |
| Left or Right | Adjust progress while the preview has focus |
| Escape | Return focus to the list, or close help |
| ? | Toggle help |
| q or Ctrl+C | Quit; when help is open, close help first |

Quit returns you to the shell. To load source changes, quit and run
`make bubblebook` again; this setup does not provide live reload.

## Add a story

Keep each component's stories beside its owning package, in a separate `stories`
subpackage. The current view stories live in
[`internal/view/stories`](../../internal/view/stories), with a separate source
and test file for each component. Keep sample data and preview-only controls
there. Production packages must not import their stories.

Add a named factory to [`catalogue.go`](catalogue.go). Each factory
must return a fresh Bubble Tea `tea.Model` with the desired initial state. The
catalogue assembles the available stories; component state and rendering belong
with the stories themselves.

Reuse components from their owning package. For a primitive that does not
implement `tea.Model`, follow the existing spinner and progress adapters: forward
its lifecycle and messages, and use the same composed rendering functions as
the application. Using a raw frame can omit styling applied by the containing
component. Do not copy the component's rendering or business logic into a story.

The gallery owns terminal input and rendering. A child must not start another
Bubble Tea program or print directly to the terminal. Preview adapters must not
invoke CLI startup, telemetry, network requests or mod operations.

Bubblebook reserves Tab, Escape, q, Ctrl+C and ? before routing keys to children.
These keys therefore cannot demonstrate a component's own input handling here.
Keep permanent-output, cancellation and terminal lifecycle verification in real
application flows; a gallery preview does not establish those contracts.

## Architecture and verification

The launcher depends on the story packages, which depend on
[shared view primitives](../../internal/view/README.md). Bubblebook registration
stays in the launcher. Application packages do not depend on stories, the gallery
or Bubblebook. This developer program is separate from the `mmm` executable and
release archives.

Follow the [terminal architecture guide](../../docs/guide-to-working-with-the-terminal.md)
when designing composed application components. The gallery provides a place to
preview those components as they become available; it does not implement the
application's target terminal session architecture.

Use the repository checks in [CONTRIBUTING.md](../../CONTRIBUTING.md). For terminal
verification tooling, see the [terminal E2E guide](../../docs/testing/terminal-harness.md).
